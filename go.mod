module github.com/hysios/edgekv

go 1.15

require (
	cuelang.org/go v0.3.2
	github.com/eclipse/paho.mqtt.golang v1.3.3
	github.com/fatih/structs v1.1.0
	github.com/go-redis/redis v6.15.7+incompatible
	github.com/go-redis/redis/v8 v8.8.2
	github.com/go-stomp/stomp v2.1.4+incompatible // indirect
	github.com/hysios/log v0.0.0-20210420091742-d54e2f0555dd
	github.com/hysios/mapindex v0.0.6
	github.com/hysios/utils v0.0.4
	github.com/kr/pretty v0.1.0
	github.com/r3labs/diff v1.1.0
	github.com/spf13/afero v1.1.2
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/buntdb v1.2.3
	github.com/tj/assert v0.0.3
	golang.org/x/exp v0.0.0-20210126221216-84987778548c
)

replace github.com/hysios/log => ../log
