jq:
	cat api/apidocs.swagger.json \
		| jq 'walk( if type == "object" then with_entries( .key |= sub( "api\\."; "") ) else . end )' \
		| jq 'walk( if type == "string" then gsub( "api."; "") else . end )' \
		> api/api.swagger.json \
	&& rm api/apidocs.swagger.json

protoc:
	protoc -I/usr/local/include \
 		-Iapi/third_party/googleapis \
 		-Iapi/ \
 		api/*.proto \
 		--go_out=plugins=grpc:api \
 		--grpc-gateway_out=logtostderr=true,allow_delete_body=true:api \
 		--swagger_out=allow_merge=true,fqn_for_swagger_name=true,allow_delete_body=true,logtostderr=true:api

openapi-generator:
	curl https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/4.2.3/openapi-generator-cli-4.2.3.jar -o openapi-generator-cli.jar

api: protoc jq

python-sdk: openapi-generator
	java -jar openapi-generator-cli.jar generate -p packageName=core.api,projectName=core.api -i api/api.swagger.json -g python -o ./sdks/python

all: api python-sdk