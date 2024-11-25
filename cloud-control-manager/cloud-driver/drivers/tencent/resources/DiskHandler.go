package resources

import (
	"errors"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type TencentDiskHandler struct {
	Region idrv.RegionInfo
	Client *cbs.Client
}

const (
	Disk_Status_Attached   = "ATTACHED"
	Disk_Status_Unattached = "UNATTACHED"
)

/*
CreateDisk 이후에 DescribeDisks 호출하여 상태가 UNATTACHED 또는 ATTACHED면 정상적으로 생성된 것임
비동기로 처리되기는 하나 생성 직후 호출해도 정상적으로 상태값을 받아옴
따라서 Operation이 완료되길 기다리는 function(WaitForXXX)은 만들지 않음
*/
func (DiskHandler *TencentDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskReqInfo.IId.NameId, "CreateDisk()")
	start := call.Start()

	existName, errExist := DiskHandler.diskExist(diskReqInfo.IId.NameId)
	if errExist != nil {
		cblogger.Error(errExist)
		return irs.DiskInfo{}, errExist
	}
	if existName {
		return irs.DiskInfo{}, errors.New("A disk with the name " + diskReqInfo.IId.NameId + " already exists.")
	}

	// region base이므로 특정 zone을 지정시 해당 zone에 생성.
	zone := DiskHandler.Region.Zone
	if diskReqInfo.Zone != "" {
		zone = diskReqInfo.Zone
	}

	request := cbs.NewCreateDisksRequest()
	request.Placement = &cbs.Placement{Zone: common.StringPtr(zone)}
	request.DiskChargeType = common.StringPtr("POSTPAID_BY_HOUR")

	var tags []*cbs.Tag
	for _, inputTag := range diskReqInfo.TagList {
		tags = append(tags, &cbs.Tag{
			Key:   common.StringPtr(inputTag.Key),
			Value: common.StringPtr(inputTag.Value),
		})
	}
	request.Tags = tags

	diskErr := validateDisk(&diskReqInfo)
	if diskErr != nil {
		cblogger.Error(diskErr)
		return irs.DiskInfo{}, diskErr
	}

	diskSize, sizeErr := strconv.ParseUint(diskReqInfo.DiskSize, 10, 64)
	if sizeErr != nil {
		return irs.DiskInfo{}, sizeErr
	}

	request.DiskSize = common.Uint64Ptr(diskSize)
	request.DiskType = common.StringPtr(diskReqInfo.DiskType)
	request.DiskName = common.StringPtr(diskReqInfo.IId.NameId)

	response, err := DiskHandler.Client.CreateDisks(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.DiskInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	newDiskId := *response.Response.DiskIdSet[0]
	cblogger.Debug(newDiskId)

	// time.Sleep(1 * time.Second)
	// 비동기로 인한 disk not found 를 보완하기 위하여 5번 retry
	retryCount := 5
	var diskFound bool
	var diskInfo irs.DiskInfo
	var diskInfoErr error
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	for i := 0; i < retryCount; i++ {
		diskInfo, diskInfoErr = DiskHandler.GetDisk(irs.IID{SystemId: newDiskId})
		if diskInfoErr != nil {
			cblogger.Debugf("Attempt [%d/5]: Checking disk status after creation. not found \n", i)

			//return irs.DiskInfo{}, diskInfoErr
			time.Sleep(1 * time.Second)
		} else {
			cblogger.Debugf("Attempt [%d/5]: Checking disk status after creation.found \n", i)
			diskFound = true
			break
		}
	}
	if !diskFound {
		cblogger.Errorf("Error during DescribeDisks after CreateDisks call: %v\n", diskInfoErr)
	}
	calllogger.Info(call.String(hiscallInfo))
	// diskInfo, diskInfoErr := DiskHandler.GetDisk(irs.IID{SystemId: newDiskId})
	// if diskInfoErr != nil {
	// 	cblogger.Error(diskInfoErr)
	// 	return irs.DiskInfo{}, diskInfoErr
	// }

	return diskInfo, nil
}

func (DiskHandler *TencentDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, "Disk", "ListDisk()")
	start := call.Start()

	diskInfoList := []*irs.DiskInfo{}

	diskSet, err := DescribeDisks(DiskHandler.Client, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	calllogger.Info(call.String(hiscallInfo))

	for _, disk := range diskSet {
		diskInfo, diskInfoErr := convertDiskInfo(disk)
		if diskInfoErr != nil {
			cblogger.Error(diskInfoErr)
			return nil, diskInfoErr
		}

		diskInfoList = append(diskInfoList, &diskInfo)
	}

	return diskInfoList, nil
}

func (DiskHandler *TencentDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "GetDisk()")
	start := call.Start()

	targetDisk, err := DescribeDisksByDiskID(DiskHandler.Client, diskIID)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.DiskInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	diskInfo, diskInfoErr := convertDiskInfo(&targetDisk)
	if diskInfoErr != nil {
		cblogger.Error(diskInfoErr)
		return irs.DiskInfo{}, diskInfoErr
	}

	return diskInfo, nil
}

func (DiskHandler *TencentDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "ChangeDiskSize()")
	start := call.Start()

	diskInfo, diskInfoErr := DiskHandler.GetDisk(diskIID)
	if diskInfoErr != nil {
		return false, diskInfoErr
	}

	diskSizeErr := validateChangeDiskSize(diskInfo, size)
	if diskSizeErr != nil {
		return false, diskSizeErr
	}

	newSize, sizeErr := strconv.ParseUint(size, 10, 64)
	if sizeErr != nil {
		cblogger.Error(sizeErr)
		return false, sizeErr
	}

	request := cbs.NewResizeDiskRequest()

	request.DiskId = common.StringPtr(diskIID.SystemId)
	request.DiskSize = common.Uint64Ptr(newSize)

	_, err := DiskHandler.Client.ResizeDisk(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	return true, nil
}

func (DiskHandler *TencentDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "DeleteDisk()")
	start := call.Start()

	request := cbs.NewTerminateDisksRequest()

	request.DiskIds = common.StringPtrs([]string{diskIID.SystemId})

	_, err := DiskHandler.Client.TerminateDisks(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	return true, nil
}

func (DiskHandler *TencentDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "AttachDisk()")
	start := call.Start()

	_, attachErr := AttachDisk(DiskHandler.Client, irs.IID{SystemId: diskIID.SystemId}, irs.IID{SystemId: ownerVM.SystemId})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if attachErr != nil {
		cblogger.Error(attachErr)
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}
	calllogger.Info(call.String(hiscallInfo))

	_, statusErr := WaitForDone(DiskHandler.Client, irs.IID{SystemId: diskIID.SystemId}, Disk_Status_Attached)
	if statusErr != nil {
		return irs.DiskInfo{}, statusErr
	}

	diskInfo, diskInfoErr := DiskHandler.GetDisk(irs.IID{SystemId: diskIID.SystemId})
	if diskInfoErr != nil {
		cblogger.Error(diskInfoErr)
		return irs.DiskInfo{}, diskInfoErr
	}

	return diskInfo, nil
}

func (DiskHandler *TencentDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "DetachDisk()")
	start := call.Start()

	request := cbs.NewDetachDisksRequest()

	request.DiskIds = common.StringPtrs([]string{diskIID.SystemId})

	_, err := DiskHandler.Client.DetachDisks(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	_, statusErr := WaitForDone(DiskHandler.Client, irs.IID{SystemId: diskIID.SystemId}, Disk_Status_Unattached)
	if statusErr != nil {
		return false, statusErr
	}

	return true, nil
}

func convertDiskInfo(diskResp *cbs.Disk) (irs.DiskInfo, error) {
	diskInfo := irs.DiskInfo{}

	diskInfo.IId = irs.IID{NameId: *diskResp.DiskName, SystemId: *diskResp.DiskId}
	diskInfo.DiskType = *diskResp.DiskType
	diskInfo.DiskSize = strconv.FormatInt(int64(*diskResp.DiskSize), 10)
	diskInfo.OwnerVM.SystemId = *diskResp.InstanceId
	diskInfo.CreatedTime, _ = time.Parse("2006-01-02 15:04:05", *diskResp.CreateTime)
	diskInfo.Status = convertTenStatusToDiskStatus(diskResp)
	diskInfo.Zone = *diskResp.Placement.Zone

	if diskResp.Tags != nil {
		var tagList []irs.KeyValue
		for _, tag := range diskResp.Tags {
			tagList = append(tagList, irs.KeyValue{
				Key:   *tag.Key,
				Value: *tag.Value,
			})
			diskInfo.TagList = tagList
		}
	}

	return diskInfo, nil
}

func convertTenStatusToDiskStatus(diskInfo *cbs.Disk) irs.DiskStatus {
	var returnStatus irs.DiskStatus

	if *diskInfo.Attached {
		returnStatus = irs.DiskAttached
	} else {
		returnStatus = irs.DiskAvailable
	}

	return returnStatus
}

func validateDisk(diskReqInfo *irs.DiskInfo) error {
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("TENCENT")
	arrDiskType := cloudOSMetaInfo.DiskType
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize
	arrRootDiskSizeOfType := cloudOSMetaInfo.RootDiskSize

	reqDiskType := diskReqInfo.DiskType
	reqDiskSize := diskReqInfo.DiskSize

	if reqDiskType == "" || reqDiskType == "default" {
		diskSizeArr := strings.Split(arrRootDiskSizeOfType[0], "|")
		reqDiskType = diskSizeArr[0]          //
		diskReqInfo.DiskType = diskSizeArr[0] // set default value
	}
	// 정의된 type인지
	if !ContainString(arrDiskType, reqDiskType) {
		return errors.New("Disktype : " + reqDiskType + "' is not valid")
	}

	if reqDiskSize == "" || reqDiskSize == "default" {
		diskSizeArr := strings.Split(arrRootDiskSizeOfType[0], "|")
		reqDiskSize = diskSizeArr[1]
		diskReqInfo.DiskSize = diskSizeArr[1] // set default value
	}

	diskSize, err := strconv.ParseInt(reqDiskSize, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	type diskSizeModel struct {
		diskType    string
		diskMinSize int64
		diskMaxSize int64
		unit        string
	}

	diskSizeValue := diskSizeModel{}
	isExists := false

	for _, diskSizeInfo := range arrDiskSizeOfType {
		diskSizeArr := strings.Split(diskSizeInfo, "|")
		if strings.EqualFold(reqDiskType, diskSizeArr[0]) {
			diskSizeValue.diskType = diskSizeArr[0]
			diskSizeValue.unit = diskSizeArr[3]
			diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}

			diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}
			isExists = true
		}
	}

	if !isExists {
		return errors.New("Invalid Disk Type : " + reqDiskType)
	}

	if diskSize < diskSizeValue.diskMinSize {
		cblogger.Error("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be at least the minimum size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
	}

	if diskSize > diskSizeValue.diskMaxSize {
		cblogger.Error("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be smaller than or equal to the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	return nil
}

func validateChangeDiskSize(diskInfo irs.DiskInfo, newSize string) error {
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("TENCENT")
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize

	diskSize, err := strconv.ParseInt(diskInfo.DiskSize, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	newDiskSize, err := strconv.ParseInt(newSize, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	if diskSize >= newDiskSize {
		return errors.New("Target Disk Size: " + newSize + " must be larger than existing Disk Size " + diskInfo.DiskSize)
	}

	type diskSizeModel struct {
		diskType    string
		diskMinSize int64
		diskMaxSize int64
		unit        string
	}

	diskSizeValue := diskSizeModel{}

	for _, diskSizeInfo := range arrDiskSizeOfType {
		diskSizeArr := strings.Split(diskSizeInfo, "|")
		if strings.EqualFold(diskInfo.DiskType, diskSizeArr[0]) {
			diskSizeValue.diskType = diskSizeArr[0]
			diskSizeValue.unit = diskSizeArr[3]
			diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}

			diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}
		}
	}

	if newDiskSize > diskSizeValue.diskMaxSize {
		cblogger.Error("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be smaller than or equal to the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	return nil
}

/*
disk가 존재하는지 check
동일이름이 없으면 false, 있으면 true
*/
func (DiskHandler *TencentDiskHandler) diskExist(chkName string) (bool, error) {
	cblogger.Debugf("chkName : %s", chkName)

	request := cbs.NewDescribeDisksRequest()

	request.Filters = []*cbs.Filter{
		{
			Name:   common.StringPtr("disk-name"),
			Values: common.StringPtrs([]string{chkName}),
		},
	}

	response, err := DiskHandler.Client.DescribeDisks(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	if *response.Response.TotalCount < 1 {
		return false, nil
	}

	cblogger.Infof("Found disk information - DiskId:[%s] / DiskName:[%s]", *response.Response.DiskSet[0].DiskId, *response.Response.DiskSet[0].DiskName)
	return true, nil
}

func (DiskHandler *TencentDiskHandler) ListIID() ([]*irs.IID, error) {
	var iidList []*irs.IID
	callLogInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, "Disk", "ListIID()")

	start := call.Start()
	diskSet, err := DescribeDisks(DiskHandler.Client, nil)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	calllogger.Debug(call.String(callLogInfo))

	for _, disk := range diskSet {
		iid := irs.IID{SystemId: *disk.DiskId}
		iidList = append(iidList, &iid)
	}
	return iidList, nil
}
