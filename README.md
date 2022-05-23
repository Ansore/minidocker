# minidocker

实现docker基本功能

- 构造容器
    - [x] 实现run命令版本容器
    - [x] 增加容器资源限制
    - [x] 增加管道以及环境变量识别

- 构建镜像
    - [x]  使用busybox创建容器
    - [x]  使用AUFS包装busybox
    - [x]  实现volume数据卷
    - [x]  实现简单镜像打包
- 构建容器进阶
    - [x]  实现后台容器运行 
    - [x]  实现查看运行后台运行中的容器 
    - [x]  实现查看容器日志 
    - [x]  实现进入容器Namespace 
    - [x]  实现停止容器 
    - [x]  实现删除容器
    - [x]  实现通过容器制作镜像 
    - [x]  实现容器指定环境变量运行 
- 容器网络
    - [x] 学习网络虚拟化技术`
    - [x]  构建容器网络模型
    - [x] 容器地址分配
    - [x] 创建Bridge网络
    - [ ] 在Bridge网络创建容器
    - [ ]  容器跨主机网络
- 高级实践
    - [ ]  使用minidocker创建一个可访问的nginx容器
    - [ ]  使用minidoker创建一个flask+redis的计数器
    - [ ]  runC
    - [ ]  runC创建容器流程
    - [ ]  Docker Containerd 项目学习
    - [ ]  Kubernetes CRI容器引擎学习
