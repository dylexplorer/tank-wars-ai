cd `dirname $0`
export GOPATH=`pwd`

export HOST=localhost:8777
export GAME=Hky4bG_lG
export SIDE=red
export TACTICS=cattycat
# export TACTICS=proxy PROXY_PORT=8776

go run src/ai-client.go > log.txt &
SIDE=blue TACTICS=cattycat go run src/ai-client.go

# run forever
# yes|while read x; do go run src/ai-client.go; done
