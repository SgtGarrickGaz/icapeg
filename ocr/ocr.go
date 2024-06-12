package ocr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/gosseract/v2"
	"gopkg.in/gographics/imagick.v2/imagick"
)

func RunOCR(clientIP string, fileName string, scannedFile []byte) bool {
	os.Mkdir("scanning/"+clientIP, os.ModePerm)
	path := filepath.Join("scanning/", clientIP, fileName)
	scanFilePath := filepath.FromSlash(path)
	scanFile, err := os.Create(scanFilePath)
	if err != nil {
		fmt.Println(err.Error())
	}
	scanFile.Write(scannedFile)

	imagick.Initialize()
	defer imagick.Terminate()
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	mw.SetResolution(150, 150)
	mw.ReadImage(scanFilePath)
	// mw.BlurImage(3, 1.2)
	mw.AdaptiveBlurImageChannel(imagick.CHANNELS_DEFAULT, 3, 1.2)
	mw.WriteImage(scanFilePath + ".png")
	client := gosseract.NewClient()
	defer client.Close()
	client.SetLanguage("eng", "hin")
	client.SetImage(scanFilePath + ".png")
	text, _ := client.Text()
	os.Remove(scanFilePath + ".png")
	os.Remove(scanFilePath)

	return strings.Contains(text, "Gandhi")
}
