package main

import (
"fmt"
"siyuan-note/siyuan/kernel/model"
"github.com/siyuan-note/logging"
"os"
)

func main() {
// Setup logging to stdout
logging.SetLevel(logging.LevelInfo)

assetPath := "/root/code/MindOcean/user-data/notes/jason/assets/test_umi_ocr.png"
if _, err := os.Stat(assetPath); err != nil {
tf("Test image not found: %v\n", err)
relative path if running from specific dir

}

fmt.Printf("Starting OCR for %s...\n", assetPath)
result, err := model.OCRAsset(assetPath)
if err != nil {
tf("OCR failed: %v\n", err)

}

fmt.Printf("OCR Success!\n")
fmt.Printf("ID: %s\n", result.ID)
fmt.Printf("FullText: %s\n", result.FullText)
fmt.Printf("PageCount: %d\n", result.PageCount)
}
