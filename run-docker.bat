set DIR=%cd%

docker run -p 1935:1935 -p 7001:7001 -p 7002:7002 -p 8090:8090 --name rtmpserver -v %DIR%/livego.yaml:/app/config/livego.yaml --restart always -d livego:latest