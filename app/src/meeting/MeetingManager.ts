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
    private intervalTimer: any = null;
    private startTime: number = 0;
    private intervalMinutes: number = 1;
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

    public async startRecording(minutes: number = 1) {
        this.intervalMinutes = minutes;
        this.countdownSec = this.intervalMinutes * 60;
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
            };

            this.source.connect(this.processor);
            this.processor.connect(this.audioContext.destination);

            this.startTime = Date.now();
            this.startTimer();
            console.log("Meeting recording started (PCM 16k), interval:", minutes, "min");
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
        console.log("Meeting recording stopped");
    }

    public get isRecording() {
        return this.audioContext?.state === "running";
    }

    public setStatusCallback(cb: (status: IMeetingStatus) => void) {
        this.statusCallback = cb;
    }

    public setInterval(minutes: number) {
        this.intervalMinutes = minutes;
        this.countdownSec = minutes * 60;
    }

    public getInterval() {
        return this.intervalMinutes;
    }

    private startTimer() {
        this.intervalTimer = setInterval(() => {
            const now = Date.now();
            const duration = Math.floor((now - this.startTime) / 1000);
            this.countdownSec--;

            if (this.countdownSec <= 0) {
                this.uploadAndTranscribe();
                this.countdownSec = this.intervalMinutes * 60;
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
        if (this.pcmBuffer.length === 0) return;

        // 1. 合并 PCM 数据并转换为 WAV 格式
        const audioBlob = this.encodeWAV(this.pcmBuffer);
        this.pcmBuffer = []; // 清空缓冲区用于下一次采集

        const formData = new FormData();
        formData.append("audio", audioBlob, `meeting_${Date.now()}.wav`);

        this._isTranscribing = true;
        console.log("Auto uploading PCM WAV for transcription...");

        fetch("/api/meeting/transcribe", {
            method: "POST",
            body: formData,
        }).then(res => res.json())
            .then(response => {
                if (response.code === 0 && response.data) {
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

        // 构建更具 AI 专业感的 HTML 内容
        const timeStr = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });

        // 使用思源内置的特定样式类（如 b3-list 等）或者自定义样式
        const content = `
<div style="margin-bottom: 16px; border: 1px solid var(--b3-border-color); border-radius: 8px; padding: 12px; background: var(--b3-theme-surface);">
    <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px; border-bottom: 1px solid var(--b3-border-color); padding-bottom: 4px;">
        <span style="font-weight: bold; color: var(--b3-theme-primary);">✨ AI 会议纪要</span>
        <span style="font-size: 12px; opacity: 0.6;">${timeStr}</span>
    </div>
    <div style="margin-bottom: 12px;">
        <div style="font-size: 12px; font-weight: bold; opacity: 0.7; margin-bottom: 4px;">🎯 核心摘要</div>
        <div style="font-size: 14px; line-height: 1.6;">${data.summary}</div>
    </div>
    <details>
        <summary style="font-size: 12px; opacity: 0.5; cursor: pointer;">查看转录原文</summary>
        <div style="font-size: 13px; opacity: 0.8; margin-top: 8px; white-space: pre-wrap;">${data.transcription}</div>
    </details>
</div>
`;

        const event = new CustomEvent("neura-meeting-transcription", { detail: content });
        window.dispatchEvent(event);
        showMessage("AI 转录已生成并插入文档", 3000);
    }
}
