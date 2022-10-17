## 分布式缓存

支持的特性:
* 单机缓存和基于HTTP的分布式缓存
* 最近最少访问(Least Recently Used, LRU)缓存策略
* 使用Go锁机制防止缓存击穿
* 使用一致性Hash选择节点, 实现负载均衡
* 使用protobuf优化节点之间二进制通信
* 使用lock-free list和分片map重构, 支持无锁高并发