@set CGO_ENABLED=0
go build -o publish/backup-uploader.exe
@set CGO_ENABLED=1
