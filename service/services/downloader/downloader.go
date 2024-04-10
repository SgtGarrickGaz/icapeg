package downloader

import (
	"fmt"
	utils "icapeg/consts"
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
	fmt.Println(d.xICAPMetadata, d.serviceName, "has started")

	if partial {
		return utils.Continue, nil, serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	file, reqContentType, err := d.generalFunc.CopyingFileToTheBuffer(d.methodName)

	if err != nil {
		fmt.Println("Error while copying file to the buffer", err.Error())
		return utils.InternalServerErrStatusCodeStr /* 500 error code */, nil, serviceHeaders,
			msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	fmt.Println("file:", file, "reqcontenttype: ", reqContentType)

	if d.httpMsg.Request.Method == http.MethodConnect {
		return utils.OkStatusCodeStr, d.generalFunc.ReturningHttpMessageWithFile(d.methodName, file.Bytes()),
			serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
	}

	return utils.NoModificationStatusCodeStr, nil, serviceHeaders, msgHeadersBeforeProcessing, msgHeadersAfterProcessing, vendorMsgs
}

func (e *Downloader) ISTagValue() string {
	epochTime := strconv.FormatInt(time.Now().Unix(), 10)
	return "epoch-" + epochTime
}
