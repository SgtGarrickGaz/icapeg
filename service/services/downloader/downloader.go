package downloader

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	utils "icapeg/consts"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"time"
)

// NewDownloaderService creates a new instance of DownloaderService

// DownloadFile downloads a file using the provided URL
func (d *Downloader) Processing(partial bool, IcapHeader textproto.MIMEHeader) (int, interface{}, map[string]string, map[string]interface{}, map[string]interface{}, map[string]interface{}) {
	/* This function returns the following types:
	   int,
	   interface{},
	   map[string]string,
	   map[string]interface{},
	   map[string]interface{},
	   map[string]interface{}
	*/
	serviceHeaders := make(map[string]string)
	serviceHeaders["X-ICAP-Metadata"] = d.xICAPMetadata
	msgHeadersBeforeProcessing := d.generalFunc.LogHTTPMsgHeaders(d.methodName)
	msgHeadersAfterProcessing := make(map[string]interface{})
	vendorMsgs := make(map[string]interface{})
	// fmt.Println(d.xICAPMetadata, d.serviceName, "has started")

	if partial {
		fmt.Println("Partial file found")
		return utils.Continue, nil, nil, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	file, _, err := d.generalFunc.CopyingFileToTheBuffer(d.methodName)

	// if d.methodName == utils.ICAPModeResp {
	// 	fmt.Println("Response Headers: ", d.httpMsg.Response.Header)
	// }

	// if d.methodName == utils.ICAPModeReq {
	// 	fmt.Println("Request Headers", d.httpMsg.Request.Header)
	// }

	if err != nil {
		fmt.Println("Error while copying file to the buffer", err.Error())
		return utils.InternalServerErrStatusCodeStr /* 500 error code */, nil, serviceHeaders,
			msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	if d.httpMsg.Request.Method == http.MethodConnect || d.httpMsg.Request.Method == http.MethodOptions {

		return utils.NoModificationStatusCodeStr, nil, serviceHeaders, msgHeadersBeforeProcessing, msgHeadersBeforeProcessing, vendorMsgs
	}

	hash := sha256.New()
	f := file
	_, err = hash.Write(f.Bytes())
	if err != nil {
		fmt.Println("Error writing the hash", err.Error())
		return utils.OkStatusCodeStr, nil, serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	fileHash := hex.EncodeToString(hash.Sum([]byte(nil)))

	if fileHash == "8edd24caa90641caecf8d4056556ba896da52ceed043e99da3071bb74d1a304d" {
		fmt.Println("Hash found")
		htmlPage, req, err := d.generalFunc.ReqModErrPage(utils.ErrPageReasonFileIsNotSafe, d.serviceName, fileHash, string(rune(file.Len())))
		if err != nil {

			return utils.InternalServerErrStatusCodeStr, nil, nil,
				msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
		}
		req.Body = io.NopCloser(htmlPage)
		msgHeadersAfterProcessing = d.generalFunc.LogHTTPMsgHeaders(d.methodName)
		return utils.OkStatusCodeStr, req, serviceHeaders,
			msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	scannedFile := f.Bytes()
	// scannedFile = d.generalFunc.PreparingFileAfterScanning(scannedFile, reqContentType, d.methodName)
	msgHeadersAfterProcessing = d.generalFunc.LogHTTPMsgHeaders(d.methodName)
	fmt.Println("fileHash:", fileHash)
	// if d.methodName == utils.ICAPModeReq {
	// 	return utils.Continue, d.httpMsg, serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	// }

	return 204, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, scannedFile), serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs

}

func (e *Downloader) ISTagValue() string {
	epochTime := strconv.FormatInt(time.Now().Unix(), 10)
	return "epoch-" + epochTime
}
