# 实现目的
深入raft算法实现，[中文文档](https://github.com/maemual/raft-zh_cn/blob/master/raft-zh_cn.md#%E5%AF%BB%E6%89%BE%E4%B8%80%E7%A7%8D%E6%98%93%E4%BA%8E%E7%90%86%E8%A7%A3%E7%9A%84%E4%B8%80%E8%87%B4%E6%80%A7%E7%AE%97%E6%B3%95%E6%89%A9%E5%B1%95%E7%89%88)，参考了[raft库](https://github.com/hashicorp/raft)，简化相关逻辑以及方便接入。

## 0.0.1
### buglist
- [x] 异常的rpc timeout 以及某个节点的高延迟问题
    * 添加内存交互，规避rpc，判断是否是rpc引起的，结果确实是rpc异常，进行进一步分析
    * net定位到waitgroup wait时候consume还未启动，直接退出了，导致的异常timeout
- [ ] 现有的日志系统会导致内存急速膨胀，急需快照优化
- [x] 更新完proto之后，出现卡死问题
    * 定位到日志模块 写lock 之后 释放的是 读unlock
- [x] vote 产生不了结果
    * 定位到follower运行时针对异常的rpc也进行了重新即时，导致自身永远不能流转candidate
    * 处理方案，对成功的rpc进行记录时间戳，这样失败的vote不影响自己的candidate排期
- [x] 当日志投递过快时候出现 miss index的问题
    * 触发删除min max时候边界处理异常

### feature
- [x] 日志同步
- [x] 内存日志优化
- [ ] 规范化启动以及api接口
- [ ] 快照
- [x] proto接入替代json
- [ ] pb扩展代码自动生成
- [ ] 支持非投票备用机
- [ ] 配置动态更新
- [ ] 客户端接入
