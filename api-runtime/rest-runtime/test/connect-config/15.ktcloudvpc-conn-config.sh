RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"ktcloudvpc-driver01","ProviderName":"KTCLOUDVPC", "DriverLibFileName":"ktcloudvpc-driver-v1.0.so"}'

 # for Cloud Credential Info
 # $$$ Need to append '/v3/' to identity_endpoint URL 
 # $$$ For 'V3' verson auth., identity_endpoint, username, password and domain_name are required basically.
 # $$$ And, need 'project_id' for the token role
 # You can get the prject id on 'Servers' > 'Token' > 'Token' menu on KT Cloud Portal
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{
    "CredentialName":"ktcloudvpc-credential01",
    "ProviderName":"KTCLOUDVPC",
    "KeyValueInfoList": [
        {"Key":"IdentityEndpoint", "Value":"https://api.ucloudbiz.olleh.com/d1/identity/v3/"},
        {"Key":"Username", "Value":"~~~@~~~.com"},
        {"Key":"Password", "Value":"XXXXXXXXXX"},
        {"Key":"DomainName", "Value":"default"},
        {"Key":"ProjectID", "Value":"XXXXXXXXXX"}
]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ktcloudvpc-DX-M1-zone","ProviderName":"KTCLOUDVPC","KeyValueInfoList": [{"Key":"Region", "Value":"KR1"}, {"Key":"Zone", "Value":"DX-M1"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ktcloudvpc-mokdong1-config","ProviderName":"KTCLOUDVPC", "DriverName":"ktcloudvpc-driver01", "CredentialName":"ktcloudvpc-credential01", "RegionName":"ktcloudvpc-DX-M1-zone"}'
