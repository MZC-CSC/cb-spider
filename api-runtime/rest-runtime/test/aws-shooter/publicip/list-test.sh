source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
	curl -X GET http://$RESTSERVER:1024/publicip?connection_name=${NAME} |json_pp &
done
