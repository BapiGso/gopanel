// Package configinit  这个包的创建是因为caddy用的glog库把-v参数占用了会导致panic
// 所以用全局变量把这个参数冲突解决掉
package configinit

import (
	"flag"
	"github.com/golang/glog"
)

// 先调用glog把flag的-v参数用掉，然后新建一个flag的命名空间test防止参数绑定冲突
var _ = func() error {
	glog.Infoln()
	//fmt.Println(flag.CommandLine.Lookup("v"), 2)
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	return nil
}()
