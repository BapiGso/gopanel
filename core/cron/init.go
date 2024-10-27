package cron

// 读取config.json文件中的所有任务，然后全部添加到cron中，每次执行cron时用viper读取自己的文件检查状态是否是暂停
//func init() {
//	list := viper.GetStringMap("cron")
//	for k1, v1 := range list {
//		for _, v2 := range v1.(map[string]map[string]any) {
//			addCron(k1, v2["script"].(string), v2["frequency"].(int), v2["attime"].(int))
//		}
//	}
//}
