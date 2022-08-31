# 一个反向代理服务器
- 利用Go httpUtil编写的反向代理服务器
- 可以通过http api进行转发服务器的动态修改
- 可以通过http api进行路由的动态设置请求头

# callback_switch
这个是已经编译好的linux应用
 - sysConf.yml配置主http接口服务
 - reverseConf.yml这是反向代理服务及路由转发配置

# maybe todo
- [ ] 负载均衡-这个得进行反向代理服务负载、压力测试后进行
- [X] 更多灵活的转发配置
- [X] 支持删除请求头
- [ ] 启动后的代理转发放在内存中，重启后就丢失了：希望进行持久化，放入redis中
- [ ] 移除IRIS的部分，更换原生的go-http服务
