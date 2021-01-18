@set CGO_ENABLED=0
go build -o publish/s3fileuploader.exe
@set CGO_ENABLED=1
