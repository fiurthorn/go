# secret

keygen:
	secret -task=keygen -r=recipients.key -i=${HOME}/identity.key

recrypt:
	cd server && secret -task=recrypt -r=../recipients.key -i=${HOME}/identity.key -confidential=path/to/file.ext.asc -confidential=../other/path/file.json.asc

master-recrypt:
	cd server && secret -task=master-recrypt -r=../recipients.key -i=../master.key -confidential=path/to/file.ext.asc -confidential=../other/path/file.json.asc

decrypt:
	cd server && secret -task=decrypt -i=${HOME}/identity.key -confidential=../other/path/file.json.asc

encrypt:
    cd server && secret -task=encrypt -r=../recipients.key -plain=path/to/file.ext -secret=path/to/file.ext.asc

build:
    cd server && secret -task=build -i=${HOME}/identity.key -secret=path/to/file.ext.asc -pkg=package_name -var=VariableName
