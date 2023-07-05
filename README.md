Fork of https://github.com/tomatocuke/openai

```sh
cp config.example.yaml config.yaml
# 修改私密信息

# local >
go run .

# remote >
# start
go build && pm2 start --name mp-go ./openai
# restart
go build && pm2 restart mp-go && pm2 log mp-go
```
