package downloader

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	utils "icapeg/consts"
	"icapeg/database"
	"icapeg/logging"
	"icapeg/ocr"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
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
	serviceHeaders["Proxy-Authorization"] = IcapHeader.Get("Proxy-Authorization")
	serviceHeaders["X-ICAP-Metadata"] = d.xICAPMetadata
	msgHeadersBeforeProcessing := d.generalFunc.LogHTTPMsgHeaders(d.methodName)
	msgHeadersAfterProcessing := make(map[string]interface{})
	vendorMsgs := make(map[string]interface{})

	if partial {
		return utils.Continue, nil, nil, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}
	// fmt.Println(d.httpMsg.Request.Header)

	// Copies the file to the buffer and gets the file size.
	file, reqContentType, err := d.generalFunc.CopyingFileToTheBuffer(d.methodName)
	fileSize := fmt.Sprintf("%v", file.Len())

	// If Error exists, return 500 status code
	if err != nil {
		fmt.Println("Error while copying file to the buffer", err.Error())
		return utils.InternalServerErrStatusCodeStr, nil, serviceHeaders,
			msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	// if Method is connect, return 204 status code.
	if d.httpMsg.Request.Method == http.MethodConnect {
		return utils.OkStatusCodeStr, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, file.Bytes()),
			serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	//Gets the client IP and checks if it is whitelisted and if true allows the request.
	ipWhiteList, err := database.NewDatabase(d.ipWhiteListDB)
	if err != nil {
		panic(err.Error())
	}

	clientIP := IcapHeader.Get("X-Client-Ip")
	whiteListed, _ := checkIPWhitelist(clientIP, ipWhiteList)
	if whiteListed {
		return utils.NoModificationStatusCodeStr, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, file.Bytes()), serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	// Gets Content Type and File Name.
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

	//  Checks the hash of the file.
	hashFile, err := database.NewDatabase(d.hashDB)

	if err != nil {
		panic(err.Error())
	}
	var isBlocked, scannerError = checkHashInFile(fileHash, hashFile)

	// If there is an error opening the hash list file
	if scannerError != nil {
		return utils.NoModificationStatusCodeStr, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, nil), serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	// If hash is found in file.
	if isBlocked {
		fmt.Println("Hash found")
		// logging.ViolationLogger.Info("Hash found: " + fileHash)

		// If watchlist, save the file to the server
		watchList, err := database.NewDatabase(d.watchListDB)
		if err != nil {
			panic(err.Error())
		}
		if checkWatchList(clientIP, watchList) {
			os.Mkdir(d.watchlistDir+"/"+clientIP, os.ModePerm)
			path := filepath.Join(d.watchlistDir, clientIP, fileName)
			newFilePath := filepath.FromSlash(path)
			newFile, err := os.Create(newFilePath)
			if err != nil {
				fmt.Println(err.Error())
			}
			newFile.Write(file.Bytes())

		}

		// If the file is an ICAP RESPMOD.
		if d.methodName == utils.ICAPModeResp {
			logging.ViolationLogger.Info("Unauthorized download: ", zap.String("client-ip", IcapHeader.Get("X-Client-Ip")), zap.String("file_hash", fileHash))

			errPage := d.generalFunc.GenHtmlPage(utils.BlockPagePath, utils.ErrPageReasonAccessProhibited, d.serviceName, fileHash, d.httpMsg.Request.RequestURI, "4096", d.xICAPMetadata)
			d.httpMsg.Response = d.generalFunc.ErrPageResp(utils.ForbiddenResourceCodeStr, errPage.Len())
			d.httpMsg.Response.Body = io.NopCloser(bytes.NewBuffer(errPage.Bytes()))
			msgHeadersAfterProcessing = d.generalFunc.LogHTTPMsgHeaders(d.methodName)
			return utils.OkStatusCodeStr, d.httpMsg.Response, serviceHeaders,
				msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs

		} else {
			logging.ViolationLogger.Info("Unauthorized upload: " + IcapHeader.Get("X-Client-Ip") + " File Hash: " + fileHash)
			htmlPage, req, err := d.generalFunc.ReqModErrPage(utils.ErrPageReasonAccessProhibited, d.serviceName, fileHash, "4096")

			if err != nil {
				fmt.Println(err.Error())
				return utils.OkStatusCodeStr, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, nil), nil,
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
	// fmt.Println("fileHash:", fileHash)

	if ocr.RunOCR(clientIP, fileName, scannedFile) {

		hashFile.WriteString(fileHash + "\n")

		if d.methodName == utils.ICAPModeResp {
			errPage := d.generalFunc.GenHtmlPage(utils.BlockPagePath, utils.ErrPageReasonAccessProhibited, d.serviceName, fileHash, d.httpMsg.Request.RequestURI, "4096", d.xICAPMetadata)
			d.httpMsg.Response = d.generalFunc.ErrPageResp(utils.ForbiddenResourceCodeStr, errPage.Len())
			d.httpMsg.Response.Body = io.NopCloser(bytes.NewBuffer(errPage.Bytes()))
			msgHeadersAfterProcessing = d.generalFunc.LogHTTPMsgHeaders(d.methodName)
			return utils.OkStatusCodeStr, d.httpMsg.Response, serviceHeaders,
				msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs

		} else {
			htmlPage, req, err := d.generalFunc.ReqModErrPage(utils.ErrPageReasonAccessProhibited, d.serviceName, fileHash, "4096")
			if err != nil {
				fmt.Println(err.Error())
				return utils.ForbiddenResourceCodeStr, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, htmlPage.Bytes()), nil,
					msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
			}
			req.Body = io.NopCloser(htmlPage)
			msgHeadersAfterProcessing = d.generalFunc.LogHTTPMsgHeaders(d.methodName)
			return utils.OkStatusCodeStr, req, serviceHeaders,
				msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
		}
	}

	return utils.NoModificationStatusCodeStr, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, scannedFile), serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs

}

func (e *Downloader) ISTagValue() string {
	epochTime := strconv.FormatInt(time.Now().Unix(), 10)
	return "epoch-" + epochTime
}

// returns true if hash found else returns false
func checkHashInFile(targetValue string, hashFile *os.File) (bool, error) {

	hashFile.Seek(0, io.SeekStart)
	scanner := bufio.NewScanner(hashFile)

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

func checkIPWhitelist(clientIP string, ipWhiteList *os.File) (bool, error) {

	ipWhiteList.Seek(0, io.SeekStart)
	scanner := bufio.NewScanner(ipWhiteList)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		trimmedLine = strings.ToLower(trimmedLine)
		clientIP = strings.ToLower(clientIP)

		// compares the two hashes and returns 1 if they match else it returns 0
		if subtle.ConstantTimeCompare([]byte(trimmedLine), []byte(clientIP)) == 1 {
			return true, nil
		}

		if err := scanner.Err(); err != nil {
			return false, err
		}

	}
	return false, nil
}

func checkWatchList(clientIP string, watchList *os.File) bool {
	watchList.Seek(0, io.SeekStart)
	scanner := bufio.NewScanner(watchList)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		trimmedLine = strings.ToLower(trimmedLine)
		clientIP = strings.ToLower(clientIP)

		// compares the two hashes and returns 1 if they match else it returns 0
		if subtle.ConstantTimeCompare([]byte(trimmedLine), []byte(clientIP)) == 1 {
			return true
		}

		if err := scanner.Err(); err != nil {
			return false
		}

	}
	return false

}
