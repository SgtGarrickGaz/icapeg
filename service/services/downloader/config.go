package downloader

import (
	http_message "icapeg/http-message"
	"icapeg/logging"
	"icapeg/readValues"
	services_utilities "icapeg/service/services-utilities"
	general_functions "icapeg/service/services-utilities/general-functions"
	"sync"
	//"time"
)

var doOnce sync.Once
var DownloaderConfig *Downloader

// Downloader represents the information regarding the Downloader service
type Downloader struct {
	xICAPMetadata string
	httpMsg       *http_message.HttpMsg
	//elapsed                    time.Duration
	serviceName                string
	methodName                 string
	watchlistDir               string
	maxFileSize                int
	bypassExts                 []string
	processExts                []string
	rejectExts                 []string
	extArrs                    []services_utilities.Extension
	returnOrigIfMaxSizeExc     bool
	return400IfFileExtRejected bool
	generalFunc                *general_functions.GeneralFunc
}

func InitDownloadConfig(serviceName string) {
	logging.Logger.Debug("loading " + serviceName + " service configurations")
	doOnce.Do(func() {
		DownloaderConfig = &Downloader{
			watchlistDir:               readValues.ReadValuesString(serviceName + ".watchlist_dir"),
			maxFileSize:                readValues.ReadValuesInt(serviceName + ".max_filesize"),
			bypassExts:                 readValues.ReadValuesSlice(serviceName + ".bypass_extensions"),
			processExts:                readValues.ReadValuesSlice(serviceName + ".process_extensions"),
			rejectExts:                 readValues.ReadValuesSlice(serviceName + ".reject_extensions"),
			returnOrigIfMaxSizeExc:     readValues.ReadValuesBool(serviceName + ".return_original_if_max_file_size_exceeded"),
			return400IfFileExtRejected: readValues.ReadValuesBool(serviceName + ".return_400_if_file_ext_rejected"),
		}
		DownloaderConfig.extArrs = services_utilities.InitExtsArr(DownloaderConfig.processExts, DownloaderConfig.rejectExts, DownloaderConfig.bypassExts)
	})
}

// NewDownloaderService returns a new populated instance of the Downloader service
func NewDownloaderService(serviceName, methodName string, httpMsg *http_message.HttpMsg, xICAPMetadata string) *Downloader {
	return &Downloader{
		xICAPMetadata:              xICAPMetadata,
		httpMsg:                    httpMsg,
		serviceName:                serviceName,
		methodName:                 methodName,
		generalFunc:                general_functions.NewGeneralFunc(httpMsg, xICAPMetadata),
		watchlistDir:               DownloaderConfig.watchlistDir,
		maxFileSize:                DownloaderConfig.maxFileSize,
		bypassExts:                 DownloaderConfig.bypassExts,
		processExts:                DownloaderConfig.processExts,
		rejectExts:                 DownloaderConfig.rejectExts,
		extArrs:                    DownloaderConfig.extArrs,
		returnOrigIfMaxSizeExc:     DownloaderConfig.returnOrigIfMaxSizeExc,
		return400IfFileExtRejected: DownloaderConfig.return400IfFileExtRejected,
	}
}
