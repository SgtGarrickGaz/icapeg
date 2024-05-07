package downloader

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	utils "icapeg/consts"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"time"
)

func (d *Downloader) Processing(partial bool, IcapHeader textproto.MIMEHeader) (int, interface{}, map[string]string, map[string]interface{}, map[string]interface{}, map[string]interface{}) {
	/* This function returns the following types:
	   int => Status Code,
	   interface{} => Http Message with File (if exists),
	   map[string]string => Service Headers
	   map[string]interface{} => Message Headers Before Processing,
	   map[string]interface{} => Message Headers After Processing,,
	   map[string]interface{} => Vendor Messages
	*/
	serviceHeaders := make(map[string]string)
	serviceHeaders["X-ICAP-Metadata"] = d.xICAPMetadata
	msgHeadersBeforeProcessing := d.generalFunc.LogHTTPMsgHeaders(d.methodName)
	msgHeadersAfterProcessing := make(map[string]interface{})
	vendorMsgs := make(map[string]interface{})
	// fmt.Println(d.xICAPMetadata, d.serviceName, "has started")

	if partial {
		fmt.Println("Partial file found")
		return utils.Continue, nil, nil, msgHeadersBeforeProcessing, msgHeadersBeforeProcessing, vendorMsgs
	}

	file, reqContentType, err := d.generalFunc.CopyingFileToTheBuffer(d.methodName)
	fileSize := fmt.Sprintf("%v", file.Len())

	if err != nil {
		fmt.Println("Error while copying file to the buffer", err.Error())
		return utils.InternalServerErrStatusCodeStr /* 500 error code */, nil, serviceHeaders,
			msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	// if method is connect, send 200 status code.
	if d.httpMsg.Request.Method == http.MethodConnect {
		return utils.OkStatusCodeStr, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, file.Bytes()), serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	var contentType []string

	var fileName string
	if d.methodName == utils.ICAPModeReq {
		contentType = d.httpMsg.Request.Header["Content-Type"]
		fileName = d.generalFunc.GetFileName()
		// fmt.Println(fileName, contentType)
	} else {
		contentType = d.httpMsg.Response.Header["Content-Type"]
		fileName = d.generalFunc.GetFileName()
		// fmt.Println(fileName, contentType)
	}
	if len(contentType) == 0 {
		contentType = append(contentType, "")
	}

	isGzip := d.generalFunc.IsBodyGzipCompressed(d.methodName)

	fileExtension := d.generalFunc.GetMimeExtension(file.Bytes(), contentType[0], fileName)

	// calculates the sha256 hash of the file
	hash := sha256.New()
	f := file
	_, err = hash.Write(f.Bytes())
	if err != nil {
		fmt.Println("Error writing the hash", err.Error())
		return utils.OkStatusCodeStr, nil, serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}
	fileHash := hex.EncodeToString(hash.Sum([]byte(nil)))

	//check if the file extension is a bypass extension
	//if yes we will not modify the file, and we will return 204 No modifications
	isProcess, icapStatus, httpMsg := d.generalFunc.CheckTheExtension(fileExtension, d.extArrs,
		d.processExts, d.rejectExts, d.bypassExts, d.return400IfFileExtRejected, isGzip,
		d.serviceName, d.methodName, fileHash, d.httpMsg.Request.RequestURI, reqContentType, file, utils.BlockPagePath, fileSize)
	if !isProcess {
		msgHeadersAfterProcessing = d.generalFunc.LogHTTPMsgHeaders(d.methodName)
		return icapStatus, httpMsg, serviceHeaders,
			msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	// compares the file hash to the given hashes
	//TODO: Implement hash lookup functionality
	var isBlocked, fileOpeningError = checkHashInFile(fileHash)
	if fileOpeningError != nil {
		return 204, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, nil), serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	if isBlocked {
		fmt.Println("Hash found")
		if d.methodName == utils.ICAPModeResp {

			errPage := d.generalFunc.GenHtmlPage("block-page.html", utils.ErrPageReasonAccessProhibited, d.serviceName, fileHash, d.httpMsg.Request.RequestURI, "0", d.xICAPMetadata)
			d.httpMsg.Response = d.generalFunc.ErrPageResp(403, errPage.Len())
			d.httpMsg.Response.Body = io.NopCloser(bytes.NewBuffer(errPage.Bytes()))

			msgHeadersAfterProcessing = d.generalFunc.LogHTTPMsgHeaders(d.methodName)
			return utils.OkStatusCodeStr, d.httpMsg.Response, serviceHeaders,
				msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs

		} else {
			htmlPage, req, err := d.generalFunc.ReqModErrPage(utils.ErrPageReasonAccessProhibited, d.serviceName, fileHash, "0")
			if err != nil {
				fmt.Println(err.Error())
				return utils.InternalServerErrStatusCodeStr, nil, nil,
					msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
			}
			req.Body = io.NopCloser(htmlPage)
			msgHeadersAfterProcessing = d.generalFunc.LogHTTPMsgHeaders(d.methodName)
			return utils.OkStatusCodeStr, req, serviceHeaders,
				msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
		}

	}

	//returns the file if other cases are false
	// default return case
	scannedFile := f.Bytes()
	msgHeadersAfterProcessing = d.generalFunc.LogHTTPMsgHeaders(d.methodName)
	fmt.Println("fileHash:", fileHash)

	return 204, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, scannedFile), serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs

}

func (e *Downloader) ISTagValue() string {
	epochTime := strconv.FormatInt(time.Now().Unix(), 10)
	return "epoch-" + epochTime
}

// returns true if hash found else returns false
func checkHashInFile(targetValue string) (bool, error) {
	file, err := os.Open("./hashlist.txt")

	if err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		trimmedLine = strings.ToLower(trimmedLine)
		targetValue = strings.ToLower(targetValue)

		// compares the two hashes and returns 1 if they match else it returns 0
		if subtle.ConstantTimeCompare([]byte(trimmedLine), []byte(targetValue)) == 1 {
			return true, nil
		}

		if err := scanner.Err(); err != nil {
			return false, err
		}

	}
	return false, nil
}
