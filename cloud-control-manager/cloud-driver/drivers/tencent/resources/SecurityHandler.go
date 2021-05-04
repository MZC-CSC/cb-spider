// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2019.06.

package resources

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type TencentSecurityHandler struct {
	Region idrv.RegionInfo
	Client *cvm.Client
}

func (securityHandler *TencentSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	return nil, nil
}

func (securityHandler *TencentSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Infof("securityNameId : [%s]", securityIID.SystemId)
	return irs.SecurityInfo{}, nil
}

func (securityHandler *TencentSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Infof("securityNameId : [%s]", securityIID.SystemId)
	return false, nil
}

/*
//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨. (보안 그룹은 그룹명으로 처리 가능하기 때문에 Name 태깅시 에러는 무시함)
//@TODO : 존재하는 보안 그룹에 정책 추가하는 기능 필요
//VPC 생략 시 활성화된 세션의 기본 VPC를 이용 함.
func (securityHandler *TencentSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Infof("securityReqInfo : ", securityReqInfo)
	spew.Dump(securityReqInfo)

	vpcId := securityReqInfo.VpcIID.SystemId

	// Create the security group with the VPC, name and description.
	//createRes, err := securityHandler.Client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
	input := ec2.CreateSecurityGroupInput{
		//GroupName:   aws.String(securityReqInfo.Name),
		GroupName: aws.String(securityReqInfo.IId.NameId),
		//Description: aws.String(securityReqInfo.Name),
		Description: aws.String(securityReqInfo.IId.NameId),
		//		VpcId:       aws.String(securityReqInfo.VpcId),awsCBNetworkInfo
		VpcId: aws.String(vpcId),
	}
	cblogger.Debugf("보안 그룹 생성 요청 정보", input)
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityReqInfo.IId.NameId,
		CloudOSAPI:   "CreateSecurityGroup()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	createRes, err := securityHandler.Client.CreateSecurityGroup(&input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidVpcID.NotFound":
				cblogger.Errorf("Unable to find VPC with ID %q.", vpcId)
				return irs.SecurityInfo{}, err
			case "InvalidGroup.Duplicate":
				cblogger.Errorf("Security group %q already exists.", securityReqInfo.IId.NameId)
				return irs.SecurityInfo{}, err
			}
		}
		cblogger.Errorf("Unable to create security group %q, %v", securityReqInfo.IId.NameId, err)
		return irs.SecurityInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Infof("[%s] 보안 그룹 생성완료", aws.StringValue(createRes.GroupId))
	spew.Dump(createRes)

	//newGroupId = *createRes.GroupId

	cblogger.Infof("인바운드 보안 정책 처리")
	//Ingress 처리
	var ipPermissions []*ec2.IpPermission
	for _, ip := range *securityReqInfo.SecurityRules {
		//for _, ip := range securityReqInfo.IPPermissions {
		if ip.Direction != "inbound" {
			cblogger.Debug("==> inbound가 아닌 보안 그룹 Skip : ", ip.Direction)
			continue
		}

		ipPermission := new(ec2.IpPermission)
		ipPermission.SetIpProtocol(ip.IPProtocol)

		if ip.FromPort != "" {
			if n, err := strconv.ParseInt(ip.FromPort, 10, 64); err == nil {
				ipPermission.SetFromPort(n)
			} else {
				cblogger.Error(ip.FromPort, "은 숫자가 아님!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetFromPort(0)
		}

		if ip.ToPort != "" {
			if n, err := strconv.ParseInt(ip.ToPort, 10, 64); err == nil {
				ipPermission.SetToPort(n)
			} else {
				cblogger.Error(ip.ToPort, "은 숫자가 아님!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetToPort(0)
		}

		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				//SetCidrIp(ip.Cidr),
				SetCidrIp("0.0.0.0/0"),
		})
		ipPermissions = append(ipPermissions, ipPermission)
	}

	//인바운드 정책이 있는 경우에만 처리
	if len(ipPermissions) > 0 {
		// Add permissions to the security group
		_, err = securityHandler.Client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			//GroupName:     aws.String(securityReqInfo.Name),
			GroupId:       createRes.GroupId,
			IpPermissions: ipPermissions,
		})
		if err != nil {
			cblogger.Errorf("Unable to set security group %q ingress, %v", securityReqInfo.IId.NameId, err)
			return irs.SecurityInfo{}, err
		}

		cblogger.Info("Successfully set security group ingress")
	}

	cblogger.Infof("아웃바운드 보안 정책 처리")
	//Egress 처리
	var ipPermissionsEgress []*ec2.IpPermission
	//for _, ip := range securityReqInfo.IPPermissionsEgress {
	for _, ip := range *securityReqInfo.SecurityRules {
		if ip.Direction != "outbound" {
			cblogger.Debug("==> outbound가 아닌 보안 그룹 Skip : ", ip.Direction)
			continue
		}

		ipPermission := new(ec2.IpPermission)
		ipPermission.SetIpProtocol(ip.IPProtocol)
		//ipPermission.SetFromPort(ip.FromPort)
		//ipPermission.SetToPort(ip.ToPort)
		if ip.FromPort != "" {
			if n, err := strconv.ParseInt(ip.FromPort, 10, 64); err == nil {
				ipPermission.SetFromPort(n)
			} else {
				cblogger.Error(ip.FromPort, "은 숫자가 아님!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetFromPort(0)
		}

		if ip.ToPort != "" {
			if n, err := strconv.ParseInt(ip.ToPort, 10, 64); err == nil {
				ipPermission.SetToPort(n)
			} else {
				cblogger.Error(ip.ToPort, "은 숫자가 아님!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetToPort(0)
		}

		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				//SetCidrIp(ip.Cidr),
				SetCidrIp("0.0.0.0/0"),
		})
		//ipPermissions = append(ipPermissions, ipPermission)
		ipPermissionsEgress = append(ipPermissionsEgress, ipPermission)
	}

	//아웃바운드 정책이 있는 경우에만 처리
	if len(ipPermissionsEgress) > 0 {

		// Add permissions to the security group
		_, err = securityHandler.Client.AuthorizeSecurityGroupEgress(&ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       createRes.GroupId,
			IpPermissions: ipPermissionsEgress,
		})
		if err != nil {
			cblogger.Errorf("Unable to set security group %q egress, %v", securityReqInfo.IId.NameId, err)
			return irs.SecurityInfo{}, err
		}

		cblogger.Info("Successfully set security group egress")
	}

	cblogger.Info("Name Tag 처리")
	//======================
	// Name 태그 처리
	//======================
	//VPC Name 태깅
	tagInput := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*createRes.GroupId),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(securityReqInfo.IId.NameId),
			},
		},
	}
	//spew.Dump(tagInput)

	_, errTag := securityHandler.Client.CreateTags(tagInput)
	//Tag 실패 시 별도의 처리 없이 에러 로그만 남겨 놓음.
	if errTag != nil {
		cblogger.Error(errTag)
	}

	//securityInfo, _ := securityHandler.GetSecurity(*createRes.GroupId)
	//securityInfo, _ := securityHandler.GetSecurity(securityReqInfo.IId) //2019-11-16 NameId 기반으로 변경됨
	securityInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: *createRes.GroupId}) //2020-04-09 SystemId기반으로 변경
	securityInfo.IId.NameId = securityReqInfo.IId.NameId                                  // Name이 필수가 아니므로 혹시 모르니 사용자가 요청한 NameId로 재설정 함.
	securityInfo.VpcIID.NameId = securityReqInfo.VpcIID.NameId                            // Name이 필수가 아니므로 객체에 저장되지 않기 때문에 시스템에서 활용 가능하도록 사용자가 요청한 NameId 값을 그대로 돌려 줌.
	return securityInfo, nil
}

func (securityHandler *TencentSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	//VPC ID 조회
	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{
			nil,
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: "List()",
		CloudOSAPI:   "DescribeSecurityGroups()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := securityHandler.Client.DescribeSecurityGroups(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info("result : ", result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		cblogger.Info("err : ", err)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	var results []*irs.SecurityInfo
	for _, securityGroup := range result.SecurityGroups {
		securityInfo := ExtractSecurityInfo(securityGroup)
		results = append(results, &securityInfo)
	}

	return results, nil
}

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
//func (securityHandler *TencentSecurityHandler) GetSecurity(securityNameId string) (irs.SecurityInfo, error) {
func (securityHandler *TencentSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Infof("securityNameId : [%s]", securityIID.SystemId)

	//2020-04-09 Filter 대신 SystemId 기반으로 변경
	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{
			aws.String(securityIID.SystemId),
		},
	}
	cblogger.Info(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityIID.SystemId,
		CloudOSAPI:   "DescribeSecurityGroups()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := securityHandler.Client.DescribeSecurityGroups(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info("result : ", result)
	cblogger.Info("err : ", err)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.SecurityInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	if len(result.SecurityGroups) > 0 {
		securityInfo := ExtractSecurityInfo(result.SecurityGroups[0])
		return securityInfo, nil
	} else {
		//return irs.SecurityInfo{}, errors.New("[" + securityNameId + "] 정보를 찾을 수 없습니다.")
		return irs.SecurityInfo{}, errors.New("InvalidSecurityGroup.NotFound: The security group '" + securityIID.SystemId + "' does not exist")
	}
}

func ExtractSecurityInfo(securityGroupResult *ec2.SecurityGroup) irs.SecurityInfo {
	var ipPermissions []irs.SecurityRuleInfo
	var ipPermissionsEgress []irs.SecurityRuleInfo
	var securityRules []irs.SecurityRuleInfo

	cblogger.Debugf("===[그룹아이디:%s]===", *securityGroupResult.GroupId)
	ipPermissions = ExtractIpPermissions(securityGroupResult.IpPermissions, "inbound")
	cblogger.Debug("InBouds : ", ipPermissions)
	ipPermissionsEgress = ExtractIpPermissions(securityGroupResult.IpPermissionsEgress, "outbound")
	cblogger.Debug("OutBounds : ", ipPermissionsEgress)
	//spew.Dump(ipPermissionsEgress)
	securityRules = append(ipPermissions, ipPermissionsEgress...)

	securityInfo := irs.SecurityInfo{
		//Id: *securityGroupResult.GroupId,
		IId: irs.IID{"", *securityGroupResult.GroupId},
		//SecurityRules: &[]irs.SecurityRuleInfo{},
		SecurityRules: &securityRules,
		VpcIID:        irs.IID{"", *securityGroupResult.VpcId},

		KeyValueList: []irs.KeyValue{
			{Key: "GroupName", Value: *securityGroupResult.GroupName},
			{Key: "VpcID", Value: *securityGroupResult.VpcId},
			{Key: "OwnerID", Value: *securityGroupResult.OwnerId},
			{Key: "Description", Value: *securityGroupResult.Description},
		},
	}

	//Name은 Tag의 "Name" 속성에만 저장됨
	cblogger.Debug("Name Tag 찾기")
	for _, t := range securityGroupResult.Tags {
		if *t.Key == "Name" {
			//securityInfo.Name = *t.Value
			securityInfo.IId.NameId = *t.Value
			cblogger.Debug("Name : ", securityInfo.IId.NameId)
			break
		}
	}

	return securityInfo
}

// IpPermission에서 공통정보 추출
func ExtractIpPermissionCommon(ip *ec2.IpPermission, securityRuleInfo *irs.SecurityRuleInfo) {
	//공통 정보
	if !reflect.ValueOf(ip.FromPort).IsNil() {
		//securityRuleInfo.FromPort = *ip.FromPort
		securityRuleInfo.FromPort = strconv.FormatInt(*ip.FromPort, 10)
	}

	if !reflect.ValueOf(ip.ToPort).IsNil() {
		//securityRuleInfo.ToPort = *ip.ToPort
		securityRuleInfo.ToPort = strconv.FormatInt(*ip.ToPort, 10)
	}

	securityRuleInfo.IPProtocol = *ip.IpProtocol
}

func ExtractIpPermissions(ipPermissions []*ec2.IpPermission, direction string) []irs.SecurityRuleInfo {
	var results []irs.SecurityRuleInfo

	for _, ip := range ipPermissions {

		//ipv4 처리
		for _, ipv4 := range ip.IpRanges {
			cblogger.Debug("Inbound/Outbound 정보 조회 : ", *ip.IpProtocol)
			securityRuleInfo := irs.SecurityRuleInfo{
				Direction: direction, // "inbound | outbound"
				//Cidr: *ipv4.CidrIp,
			}
			cblogger.Debug(*ipv4.CidrIp)

			ExtractIpPermissionCommon(ip, &securityRuleInfo) //IP & Port & Protocol 추출
			results = append(results, securityRuleInfo)
		}

		//ipv6 처리
		for _, ipv6 := range ip.Ipv6Ranges {
			securityRuleInfo := irs.SecurityRuleInfo{
				Direction: direction, // "inbound | outbound"
				//Cidr: *ipv6.CidrIpv6,
			}
			cblogger.Debug(*ipv6.CidrIpv6)

			ExtractIpPermissionCommon(ip, &securityRuleInfo) //IP & Port & Protocol 추출
			results = append(results, securityRuleInfo)
		}

		//ELB나 보안그룹 참조 방식 처리
		for _, userIdGroup := range ip.UserIdGroupPairs {
			securityRuleInfo := irs.SecurityRuleInfo{
				Direction: direction, // "inbound | outbound"
				//Cidr: *userIdGroup.GroupId,
			}
			cblogger.Debug(*userIdGroup.UserId)

			ExtractIpPermissionCommon(ip, &securityRuleInfo) //IP & Port & Protocol 추출
			results = append(results, securityRuleInfo)
		}
	}

	return results
}

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
//func (securityHandler *TencentSecurityHandler) DeleteSecurity(securityNameId string) (bool, error) {
func (securityHandler *TencentSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Infof("securityNameId : [%s]", securityIID.SystemId)

	//securityID := securityInfo.Id
	//securityID := securityInfo.IId.SystemId
	securityID := securityIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityIID.SystemId,
		CloudOSAPI:   "DeleteSecurityGroup()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	// Delete the security group.
	_, err := securityHandler.Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(securityID),
	})
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroupId.Malformed":
				fallthrough
			case "InvalidGroup.NotFound":
				cblogger.Errorf("%s.", aerr.Message())
				return false, err
			}
		}
		cblogger.Errorf("Unable to get descriptions for security groups, %v.", err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Successfully delete security group %q.", securityID)

	return true, nil
}
*/
