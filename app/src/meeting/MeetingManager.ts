import { fetchSyncPost } from "../util/fetch";
import { showMessage } from "../dialog/message";

export interface IMeetingStatus {
    isRecording: boolean;
    isTranscribing: boolean;
    duration: number;
    nextUploadCountdown: number;
}

export class MeetingManager {
    private static instance: MeetingManager;
    private audioContext: AudioContext | null = null;
    private processor: ScriptProcessorNode | null = null;
    private source: MediaStreamAudioSourceNode | null = null;
    private pcmBuffer: Float32Array[] = [];
    private audioProcessCount: number = 0;
    private lastAudioCheck: number = 0;
    private totalSamplesCollected: number = 0;
    private intervalTimer: any = null;
    private startTime: number = 0;
    private intervalSeconds: number = 60;
    private statusCallback: ((status: IMeetingStatus) => void) | null = null;
    private countdownSec: number = 0;
    private _isTranscribing: boolean = false;
    private stream: MediaStream | null = null;

    private constructor() { }

    public static getInstance() {
        if (!MeetingManager.instance) {
            MeetingManager.instance = new MeetingManager();
        }
        return MeetingManager.instance;
    }

    public async startRecording(seconds: number = 60) {
        this.intervalSeconds = seconds;
        this.countdownSec = this.intervalSeconds;
        this.pcmBuffer = [];

        try {
            this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });

            // 关键：强制设置采样率为 16000，解决 ASR 识别率问题
            this.audioContext = new (window.AudioContext || (window as any).webkitAudioContext)({ sampleRate: 16000 });
            this.source = this.audioContext.createMediaStreamSource(this.stream);

            // 使用 ScriptProcessorNode 收集原始 PCM 数据
            this.processor = this.audioContext.createScriptProcessor(4096, 1, 1);

            this.processor.onaudioprocess = (e) => {
                const inputData = e.inputBuffer.getChannelData(0);
                // 深度拷贝数据
                this.pcmBuffer.push(new Float32Array(inputData));

                // 诊断：统计收集的音频数据
                this.audioProcessCount++;
                this.totalSamplesCollected += inputData.length;

                // 每 100 次打印一次诊断信息
                if (this.audioProcessCount % 100 === 0) {
                    console.log("Audio process diagnostic:", {
                        processCount: this.audioProcessCount,
                        bufferChunks: this.pcmBuffer.length,
                        totalSamples: this.totalSamplesCollected,
                        estimatedDuration: (this.totalSamplesCollected / 16000).toFixed(2) + "秒"
                    });
                }
            };

            this.source.connect(this.processor);
            this.processor.connect(this.audioContext.destination);

            this.startTime = Date.now();
            this.startTimer();
            console.log("Meeting recording started (PCM 16k), interval:", seconds, "s");
        } catch (err) {
            showMessage("无法访问麦克风: " + err);
            throw err;
        }
    }

    public stopRecording() {
        this.clearTimer();
        if (this.processor) {
            this.processor.onaudioprocess = null;
            this.processor.disconnect();
            this.processor = null;
        }
        if (this.source) {
            this.source.disconnect();
            this.source = null;
        }
        if (this.audioContext) {
            this.audioContext.close();
            this.audioContext = null;
        }
        if (this.stream) {
            this.stream.getTracks().forEach(track => track.stop());
            this.stream = null;
        }

        if (this.statusCallback) {
            this.statusCallback({
                isRecording: false,
                isTranscribing: false,
                duration: 0,
                nextUploadCountdown: 0
            });
        }

        console.log("Meeting recording stopped");
    }

    public get isRecording() {
        return this.audioContext?.state === "running";
    }

    public setStatusCallback(cb: (status: IMeetingStatus) => void) {
        this.statusCallback = cb;
    }

    public setInterval(seconds: number) {
        this.intervalSeconds = seconds;
        this.countdownSec = seconds;
    }

    public getInterval() {
        return this.intervalSeconds;
    }

    private startTimer() {
        this.intervalTimer = setInterval(() => {
            const now = Date.now();
            const duration = Math.floor((now - this.startTime) / 1000);
            this.countdownSec--;

            if (this.countdownSec <= 0) {
                this.uploadAndTranscribe();
                this.countdownSec = this.intervalSeconds;
            }

            if (this.statusCallback) {
                this.statusCallback({
                    isRecording: true,
                    isTranscribing: this._isTranscribing,
                    duration: duration,
                    nextUploadCountdown: this.countdownSec
                });
            }
        }, 1000);
    }

    private clearTimer() {
        if (this.intervalTimer) {
            clearInterval(this.intervalTimer);
            this.intervalTimer = null;
        }
    }

    public async uploadAndTranscribe() {
        if (this.pcmBuffer.length === 0) {
            console.warn("No PCM data to upload, skipping transcription");
            return;
        }

        // 1. 合并 PCM 数据并转换为 WAV 格式
        const audioBlob = this.encodeWAV(this.pcmBuffer);

        // 诊断：计算音频数据的详细信息
        const totalSamples = this.pcmBuffer.reduce((acc, s) => acc + s.length, 0);
        const estimatedDuration = (totalSamples / 16000).toFixed(2);

        // 检查音量 (音量过小可能是麦克风问题或授权失效)
        let maxAmp = 0;
        for (const chunk of this.pcmBuffer) {
            for (let i = 0; i < chunk.length; i++) {
                const a = Math.abs(chunk[i]);
                if (a > maxAmp) maxAmp = a;
            }
        }

        console.log("Audio encoding completed:", {
            maxAmplitude: maxAmp.toFixed(4),
            blobSize: audioBlob.size,
            totalSamples: totalSamples,
            estimatedDuration: estimatedDuration + "秒"
        });

        if (maxAmp < 0.01) {
            console.warn("Detected very low audio amplitude. Mic might be muted or not working.");
        }

        this.pcmBuffer = []; // 清空缓冲区用于下一次采集

        const formData = new FormData();
        formData.append("audio", audioBlob, `meeting_${Date.now()}.wav`);

        this._isTranscribing = true;
        console.log("Uploading audio for transcription...");

        fetch("/api/meeting/transcribe", {
            method: "POST",
            body: formData,
        }).then(res => res.json())
            .then(response => {
                console.log("Transcription API response:", response);
                if (response.code === 0 && response.data) {
                    console.log("Transcription successful:", {
                        transcription: response.data.transcription,
                        summary: response.data.summary
                    });
                    this.insertTranscriptionToEditor(response.data);
                } else {
                    console.error("Transcription failed:", response.msg);
                    showMessage("转录失败: " + response.msg);
                }
            }).catch(err => {
                console.error("Upload error:", err);
                showMessage("转录上传失败，请检查网络");
            }).finally(() => {
                this._isTranscribing = false;
            });
    }

    private encodeWAV(samples: Float32Array[]) {
        const sampleRate = 16000;
        const totalLength = samples.reduce((acc, s) => acc + s.length, 0);
        const buffer = new ArrayBuffer(44 + totalLength * 2);
        const view = new DataView(buffer);

        const writeString = (offset: number, string: string) => {
            for (let i = 0; i < string.length; i++) {
                view.setUint8(offset + i, string.charCodeAt(i));
            }
        };

        // WAV 头部信息
        writeString(0, 'RIFF');
        view.setUint32(4, 32 + totalLength * 2, true);
        writeString(8, 'WAVE');
        writeString(12, 'fmt ');
        view.setUint32(16, 16, true);
        view.setUint16(20, 1, true); // PCM 格式
        view.setUint16(22, 1, true); // 单声道
        view.setUint32(24, sampleRate, true);
        view.setUint32(28, sampleRate * 2, true); // Byte rate
        view.setUint16(32, 2, true); // Block align
        view.setUint16(34, 16, true); // Bits per sample
        writeString(36, 'data');
        view.setUint32(40, totalLength * 2, true);

        // 写入 PCM 采样数据
        let offset = 44;
        for (let i = 0; i < samples.length; i++) {
            const sample = samples[i];
            for (let j = 0; j < sample.length; j++) {
                // 将 Float32 (-1.0 到 1.0) 转换为 Int16 (-32768 到 32767)
                const s = Math.max(-1, Math.min(1, sample[j]));
                view.setInt16(offset, s < 0 ? s * 0x8000 : s * 0x7FFF, true);
                offset += 2;
            }
        }

        return new Blob([buffer], { type: 'audio/wav' });
    }

    private insertTranscriptionToEditor(data: { transcription: string, summary: string }) {
        if (!data.transcription) return;

        // 严格过滤大模型的思考过程（包括未闭合的标签）
        let cleanSummary = data.summary
            .replace(/<think>[\s\S]*?<\/think>/gi, "")
            .replace(/<think>[\s\S]*/gi, "")
            .replace(/<\/think>/gi, "")
            .trim();
        const cleanTranscription = data.transcription
            .replace(/<think>[\s\S]*?<\/think>/gi, "")
            .replace(/<think>[\s\S]*/gi, "")
            .trim();

        // 解析会议纪要三行内容
        const parsedSummary = this.parseMeetingSummary(cleanSummary);

        // 构建紧凑的思源笔记块格式
        const timeStr = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });

        // 使用思源笔记原生块格式，更紧凑
        const content = `> ✨ **AI 纪要** <span style="opacity: 0.6; font-size: 0.9em;">${timeStr}</span>
> 📌 **主题**：${parsedSummary.theme}
> 💬 **要点**：${parsedSummary.discussion}
> ⚡ **后续**：${parsedSummary.actions}

> 🔽 **转录原文**（点击展开）
> ${cleanTranscription.split('\n').map(line => '> ' + line).join('\n')}
`;

        const event = new CustomEvent("neura-meeting-transcription", { detail: content });
        window.dispatchEvent(event);
        showMessage("AI 转录已实时同步");
    }

    /**
     * 解析会议纪要三行内容为结构化数据
     */
    private parseMeetingSummary(summary: string): { theme: string, discussion: string, actions: string } {
        // 默认值
        const result = {
            theme: "未提取到主题",
            discussion: "未提取到要点",
            actions: "未提取到后续"
        };

        if (!summary) return result;

        // 按行分割并清理
        const lines = summary.split('\n').map(line => line.trim()).filter(line => line.length > 0);

        // 解析每一行，移除 markdown 标记
        for (const line of lines) {
            const cleanLine = line
                .replace(/^\s*[-*>]+\s*/g, '')  // 移除列表标记和引号
                .replace(/^\*\*[^*]+\*\*[:：]?\s*/g, '')  // 移除 **主题：** 这样的前缀
                .trim();

            // 匹配第一行（主题/会议主题）
            if (line.includes('主题') || line.includes('会议主题')) {
                result.theme = cleanLine || result.theme;
            }
            // 匹配第二行（要点/讨论/关键讨论）
            else if (line.includes('要点') || line.includes('讨论') || line.includes('关键讨论')) {
                result.discussion = cleanLine || result.discussion;
            }
            // 匹配第三行（后续/行动/行动项/决议）
            else if (line.includes('后续') || line.includes('行动') || line.includes('行动项') || line.includes('决议') || line.includes('结论')) {
                result.actions = cleanLine || result.actions;
            }
        }

        // 如果没有匹配到特定格式，按顺序分配
        if (result.theme === "未提取到主题" && lines.length > 0) {
            result.theme = lines[0].replace(/^\s*[-*>]+\s*/g, '').replace(/^\*\*[^*]+\*\*[:：]?\s*/g, '').trim() || lines[0];
        }
        if (result.discussion === "未提取到要点" && lines.length > 1) {
            result.discussion = lines[1].replace(/^\s*[-*>]+\s*/g, '').replace(/^\*\*[^*]+\*\*[:：]?\s*/g, '').trim() || lines[1];
        }
        if (result.actions === "未提取到后续" && lines.length > 2) {
            result.actions = lines[2].replace(/^\s*[-*>]+\s*/g, '').replace(/^\*\*[^*]+\*\*[:：]?\s*/g, '').trim() || lines[2];
        }

        return result;
    }
}
