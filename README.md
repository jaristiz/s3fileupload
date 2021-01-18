# s3fileupload
File uploader to S3 buckets

## Compilation
To compile run the command `./build.sh` from a command prompt which will store the result to the `publish` folder.

## Executing
To execute you need to provide the required information as environment variables, you can do that as shown below:

```
set XUPLOADERID={Your Aws AccessKeyID}
set XUPLOADERKEY={Your Aws SecretAccessKey }
set XUPLOADERDIRECTORY={Your S3 Bucket Name}
....
s3fileuploader
```
