- [x] 异常的rpc timeout 以及某个节点的高延迟问题
    * 添加内存交互，规避rpc，判断是否是rpc引起的，结果确实是rpc异常，进行进一步分析
    * net定位到waitgroup wait时候consume还未启动，直接退出了，导致的异常timeout

