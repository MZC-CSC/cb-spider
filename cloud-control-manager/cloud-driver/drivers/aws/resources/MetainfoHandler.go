package resources

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
)

type AwsMetainfoHandler struct {
	Region         idrv.RegionInfo
	Client         *ec2.EC2
}


// 모든 리전(권한이 없어도)을 조회하여 리스트로 가져와 AvailabilityZone을 리전당 조회합니다. 
func (metaInfoHandler *AwsMetainfoHandler) GetAllRegionZone() ([]ec2.DescribeAvailabilityZonesOutput,error){
	cblogger.Info("AWS Driver: called Metainfo()/GetAllRegionZone()!")

	hiscallInfo := GetCallLogScheme(metaInfoHandler.Region, call.METAINFO, "Meta", "GetAllRegionZone()")
	start := call.Start()

	regionsInput := &ec2.DescribeRegionsInput{
		AllRegions : aws.Bool(true),
	}
	regionReq, regionResp := metaInfoHandler.Client.DescribeRegionsRequest(regionsInput)
	regionErr := regionReq.Send()
	if regionErr != nil {
		return nil, regionErr
	}

	Zoneinput := &ec2.DescribeAvailabilityZonesInput{
		AllAvailabilityZones : aws.Bool(true),
	}
	var availabilityZones []ec2.DescribeAvailabilityZonesOutput
	for _, region := range regionResp.Regions {
		metaInfoHandler.Client.Client.Config.Region = region.RegionName
		zoneReq, zoneResp := metaInfoHandler.Client.DescribeAvailabilityZonesRequest(Zoneinput)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		zoneErr := zoneReq.Send()
		if zoneErr != nil {
			return nil, zoneErr
		}
		availabilityZones = append(availabilityZones, *zoneResp)
	}

	calllogger.Info(call.String(hiscallInfo))

	return availabilityZones, nil
}
