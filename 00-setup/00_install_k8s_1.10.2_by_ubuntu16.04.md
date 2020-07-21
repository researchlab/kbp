1.下载ubuntu img

    wget -c https://mirrors.tuna.tsinghua.edu.cn/ubuntu-releases/16.04.6/ubuntu-16.04.6-server-amd64.iso

2.在ubuntu 系统上添加docker 公钥

    curl -fsSL https://download.docker.com/linux/ubuntu/gpg |apt-key add -

3.添加docker 官方源仓库

    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable”

4.更新本地包索引

    apt-get update
5.查询

    apt-cache madison docker-ce 

6.安装对应版本

    apt-get install docker-ce=17.03.2~ce-0~ubuntu-xenial
7.查看安装情况

    docker version 
    systemctl status docker 

8.添加k8s 公钥 

直接在vm中执行如下命令多半是失败的, 如下,

    root@k8s1:~# curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg |apt-key add -
    gpg: no valid OpenPGP data found.

可以在宿主机上开启vpn ,然后在宿主机上下载完成， 然后scp 到 vm中，

    ➜  /tmp scp -P 9091 apt-key.gpg root@127.0.0.1:~/
    apt-key.gpg                                                                                                  100%  653   677.7KB/s   00:00
> 上面 scp 通过 -P 指定端口号，且 -P 紧随scp命令才可;

本地添加k8s 公钥
```
root@k8s1:~# ls
apt-key.gpg
root@k8s1:~# cat apt-key.gpg |apt-key add -
OK
```

9.创建k8s list 文件

    vi /etc/apt/sources.list.d/kubernetes.list
    deb http://apt.kubernetes.io/ kubernetes-xenial main

更新本地包缓存
    
    apt-get update 

10.安装k8s 工具

    apt-get install -y kubelet=1.10.2-00  kubeadm=1.10.2-00 kubectl=1.10.2-00 kubernetes-cni=0.6.0-00 --allow-downgrades
> 需要指定版本1.10.2-00

11.检查安装情况
```
Setting up kubelet (1.10.2-00) ...
Setting up kubectl (1.10.2-00) ...
Setting up kubeadm (1.10.2-00) ...
root@k8s1:~# systemctl status kubelet
```
> 此时的kubelet 还不能正常启动，因为环境参数不满足;

核心组件镜像下载

> kubeadm config images list 获取当前要部署的核心组件库列表

1.kubeadm 默认从k8s.gcr.io上下载核心组件镜像;

方式一: k8s 1.8之前可以通过KUBE_REPO_PREFIX 指定其它k8s镜像仓库前缀 使用其它仓库下载; k8s 1.8之后就不行了， 只能通过配置加速器;

方式二: 手工将镜像下载到本地进行导入安装

images
```
k8s.gcr.io/kube-apiserver-${ARCH}
k8s.gcr.io/kube-controller-manager-${ARCH}
k8s.gcr.io/kube-scheduler-${ARCH}
k8s.gcr.io/kube-proxy-${ARCH}
k8s.gcr.io/etcd-${ARCH}
k8s.gcr.io/pause-${ARCH}
k8s.gcr.io/kube-dns-sidecar-${ARCH}
k8s.gcr.io/kube-dns-kube-dns-${ARCH}
k8s.gcr.io/kube-dns-dnsmasq-nanny-${ARCH}
```

k8s master 节点要下载的核心组件

cat master-images-list.txt
```
docker pull anjia0532/kube-apiserver-amd64:v1.10.2
docker pull anjia0532/kube-controller-manager-amd64:v1.10.2
docker pull anjia0532/kube-scheduler-amd64:v1.10.2
docker pull anjia0532/kube-proxy-amd64:v1.10.2
docker pull anjia0532/etcd-amd64:3.1.12
docker pull anjia0532/pause-amd64:3.1
docker pull anjia0532/k8s-dns-sidecar-amd64:1.14.8
docker pull anjia0532/k8s-dns-kube-dns-amd64:1.14.8
docker pull anjia0532/k8s-dns-dnsmasq-nanny-amd64:1.14.8
```

快捷脚本
```
chmod +x kubeadm_config_images_list.sh
#! /bin/bash
images=(
    kube-apiserver:v1.12.2
    kube-controller-manager:v1.12.2
    kube-scheduler:v1.12.2
    kube-proxy:v1.12.2
    pause:3.1
    etcd:3.2.24
    coredns:1.2.2
)
 
for imageName in ${images[@]} ; do
    docker pull registry.cn-hangzhou.aliyuncs.com/google_containers/$imageName
    docker tag registry.cn-hangzhou.aliyuncs.com/google_containers/$imageName k8s.gcr.io/$imageName
 
done
```

给虚拟机配置宿主机vpn 

    root@k8s1:~# export https_proxy=http://10.21.71.233:7890 http_proxy=http://10.21.71.233:7890 all_proxy=socks5://10.21.71.233:7891
> 10.21.71.233 是宿主机ip  7890 是宿主机vpn 端口

将下载好的镜像重新打标签
```
docker tag anjia0532/kube-apiserver-amd64:v1.10.2 k8s.gcr.io/kube-apiserver-amd64:v1.10.2
docker tag anjia0532/kube-scheduler-amd64:v1.10.2 k8s.gcr.io/kube-scheduler-amd64:v1.10.2
docker tag anjia0532/kube-controller-manager-amd64:v1.10.2 k8s.gcr.io/kube-controller-manager-amd64:v1.10.2
docker tag anjia0532/kube-proxy-amd64:v1.10.2 k8s.gcr.io/kube-proxy-amd64:v1.10.2
docker tag anjia0532/etcd-amd64:3.1.12 k8s.gcr.io/etcd-amd64:3.1.12
docker tag anjia0532/pause-amd64:3.1 k8s.gcr.io/pause-amd64:3.1
docker tag anjia0532/k8s-dns-sidecar-amd64:1.14.8 k8s.gcr.io/k8s-dns-sidecar-amd64:1.14.8
docker tag anjia0532/k8s-dns-kube-dns-amd64:1.14.8 k8s.gcr.io/k8s-dns-kube-dns-amd64:1.14.8
docker tag anjia0532/k8s-dns-dnsmasq-nanny-amd64:1.14.8 k8s.gcr.io/k8s-dns-dnsmasq-nanny-amd64:1.14.8
```

k8s-worker1,2 节点 只需要下载下面两个镜像即可
```
docker pull anjia0532/kube-proxy-amd64:v1.10.2
docker pull anjia0532/pause-amd64:3.1
```

初始化master 节点

前置步骤:  选择网络插件， 这里选择Weave Net

实际操作:  执行 kubeadm init 

    kubeadm init —apiserver-advertise-address=10.0.2.5  —pod-network-cidr=192.168.16.0/20   
> 这里10.0.2.5 是 k8s master 节点ip地址
```
root@k8s1:~# kubeadm init --apiserver-advertise-address=10.0.2.4 --pod-network-cidr=192.168.16.0/20
[init] Using Kubernetes version: v1.10.13
[init] Using Authorization modes: [Node RBAC]
[preflight] Running pre-flight checks.
[preflight] Some fatal errors occurred:
    [ERROR Swap]: running with swap on is not supported. Please disable swap
[preflight] If you know what you are doing, you can make a check non-fatal with `--ignore-preflight-errors=…`
```

不支持 swap, 需禁用swap 

    vim /etc/fstabe 注释最后一行 然后重启 reboot now 

再次执行
```
root@k8s1:~# kubeadm reset
root@k8s1:~# kubeadm init --apiserver-advertise-address=10.0.2.4 --pod-network-cidr=192.168.16.0/20  —kubernetes-version=1.10.2 
```
> 加上—kubernetes-version=1.10.2   指定k8s 版本
> 如果中途出错可以用kubeadm reset来进行回退

安装完成提示如下, 安装成功
```
[addons] Applied essential addon: kube-dns
[addons] Applied essential addon: kube-proxy

Your Kubernetes master has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
  https://kubernetes.io/docs/concepts/cluster-administration/addons/

You can now join any number of machines by running the following on each node
as root:

  kubeadm join 10.0.2.4:6443 --token qyg66u.hn2ohnm0hl3e8frh --discovery-token-ca-cert-hash sha256:4cc08f46b1e1b663c9127bace7cb1fe5d3b03437ecd5899a22c30fbaec1a3984
```


设置env 变量
```
root@k8s1:~# export KUBECONFIG=/etc/kubernetes/admin.conf
kubectl get pods -n kube-system -o wide 

root@k8s1:~# kubectl get pods -n kube-system -o wide
NAME                           READY     STATUS    RESTARTS   AGE       IP         NODE
etcd-k8s1                      1/1       Running   0          6m        10.0.2.4   k8s1
kube-apiserver-k8s1            1/1       Running   0          6m        10.0.2.4   k8s1
kube-controller-manager-k8s1   1/1       Running   0          6m        10.0.2.4   k8s1
kube-dns-86f4d74b45-4262p      0/3       Pending   0          6m        <none>     <none>
kube-proxy-fd7gc               1/1       Running   0          6m        10.0.2.4   k8s1
kube-scheduler-k8s1            1/1       Running   0          6m        10.0.2.4   k8s1
root@k8s1:~#
```

可以看到只有dns没有起来，  这个需要安装完网络插件后即可Running

安装Weave Net 插件

下载weave.yaml 文件

    curl -L "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version| base64 | tr -d '\n')" > weave.yaml

修改weave.yaml  网络地址范围， 在env下添加下面的ip 范围
```
 - name: IPALLOC_RANGE
  value: 192.168.16.0/20
```
```
root@k8s1:~# kubectl apply -f weave.yaml
serviceaccount "weave-net" created
clusterrole.rbac.authorization.k8s.io "weave-net" created
clusterrolebinding.rbac.authorization.k8s.io "weave-net" created
role.rbac.authorization.k8s.io "weave-net" created
rolebinding.rbac.authorization.k8s.io "weave-net" created
daemonset.apps "weave-net” created
root@k8s1:~# kubectl get po -n kube-system
NAME                           READY     STATUS    RESTARTS   AGE
etcd-k8s1                      1/1       Running   0          19m
kube-apiserver-k8s1            1/1       Running   0          20m
kube-controller-manager-k8s1   1/1       Running   0          19m
kube-dns-86f4d74b45-4262p      3/3       Running   0          20m
kube-proxy-fd7gc               1/1       Running   0          20m
kube-scheduler-k8s1            1/1       Running   0          19m
weave-net-7dch9                2/2       Running   0          5m
```

将worker节点加入集群

通过 kubectl get nodes 查看集群信息

在worker节点执行如下信息，
```
root@k8s2:/etc# kubeadm join 10.0.2.4:6443 --token qyg66u.hn2ohnm0hl3e8frh --discovery-token-ca-cert-hash sha256:4cc08f46b1e1b663c9127bace7cb1fe5d3b03437ecd5899a22c30fbaec1a3984
[preflight] Running pre-flight checks.
    [WARNING Hostname]: hostname "k8s2" could not be reached
    [WARNING Hostname]: hostname "k8s2" lookup k8s2 on 202.96.209.133:53: no such host
[preflight] Some fatal errors occurred:
    [ERROR CRI]: unable to check if the container runtime at "/var/run/dockershim.sock" is running: fork/exec /usr/bin/crictl -r /var/run/dockershim.sock info: no such file or directory
[preflight] If you know what you are doing, you can make a check non-fatal with `--ignore-preflight-errors=...`
root@k8s2:/etc# rm -f /usr/bin/crictl
root@k8s2:/etc# kubeadm join 10.0.2.4:6443 --token qyg66u.hn2ohnm0hl3e8frh --discovery-token-ca-cert-hash sha256:4cc08f46b1e1b663c9127bace7cb1fe5d3b03437ecd5899a22c30fbaec1a3984
```
>  报错， 执行rm -f /usr/bin/crictl，再次执行join即可;

此时在master 节点k8s1 中可以再次查看集群信息 发现  k8s2 已经加入进来了
```
root@k8s1:~# kubectl get nodes
NAME      STATUS    ROLES     AGE       VERSION
k8s1      Ready     master    36m       v1.10.2
k8s2      Ready     <none>    1m        v1.10.2
```

配置完成之后， 工作负载都会打到woker节点上， 如果想让master节点也承载工作负载可执行如下命令， 
让Master 节点承载工作负载(可选)  

    kubectl taint nodes —all node-role.kubernetes.io/master-

> 但是在生产环境中 建议不让master节点承载工作负载; 

目前只能在master节点上使用 kubectl , 如何可以在worker节点也是用kubectl 命令?

将master 节点中的 /etc/kubernetes/admin.conf   配置 scp 到worker节点上去；

    root@k8s1:~# scp /etc/kubernetes/admin.conf root@10.0.2.6:/etc/kubernetes/

编辑.bashrc 
```
root@k8s2:/etc/kubernetes# cat ~/.bashrc
export KUBECONFIG=/etc/kubernetes/admin.conf
source ~/.bashrc 
```

安装Dashboard
```
docker pull anjia0532/kubernetes-dashboard-amd64:v1.8.3
root@k8s1:~# docker tag anjia0532/kubernetes-dashboard-amd64:v1.8.3 k8s.gcr.io/kubernetes-dashboard-amd64:v1.8.3
root@k8s1:~# wget https://raw.githubusercontent.com/kubernetes/dashboard/v1.10.1/src/deploy/recommended/kubernetes-dashboard.yaml
root@k8s1:~# wget https://raw.githubusercontent.com/kubernetes/dashboard/v1.8.3/src/deploy/recommended/kubernetes-dashboard.yaml
```
> 注意这里选择  v1.8.3 的 dashboard 

```
root@k8s1:~# kubectl apply -f kubernetes-dashboard.yaml
secret "kubernetes-dashboard-certs" unchanged
serviceaccount "kubernetes-dashboard" unchanged
role.rbac.authorization.k8s.io "kubernetes-dashboard-minimal" unchanged
rolebinding.rbac.authorization.k8s.io "kubernetes-dashboard-minimal" unchanged
deployment.apps "kubernetes-dashboard" created
service "kubernetes-dashboard" created
```
```
root@k8s1:~# kubectl get po -n kube-system -o wide |grep dashboard
kubernetes-dashboard-7d5dcdb6d9-4hq6g   1/1       Running   0          6s        192.168.28.1   k8s2
root@k8s1:~#
```

用k8s proxy 暴露端口8009

    root@k8s1:~# kubectl proxy --address=0.0.0.0 --port=8009

然后设置virtualbox 全局适配器  端口转发, 这里转发到mac 宿主机为9094端口， 然后通过如下地址打开kubernetes-dashboard 页面; 

    http://127.0.0.1:9094/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/#!/login


创建用户
```
root@k8s1:~# cat kubernetes-adminuser.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: admin-user
  namespace: kube-system

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: admin-user
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: admin-user
  namespace: kube-system
root@k8s1:~# kubectl apply -f kubernetes-adminuser.yaml
```

获取令牌

    root@k8s1:~# kubectl -n kube-system describe secret $(kubectl -n kube-system get secret | grep admin-user | awk '{print $1}’)
		# or
		root@k8s1:~# kubectl get secret -n kube-system |grep admin-user-token
		root@k8s1:~# kubectl describe secret/admin-user-token-wq4nw -n kube-system

> 注意新版的k8s dashboard 是装在  kubernetes-dashboard 空间上的， 上面的是装在kube-system 空间上的;

virtualbox 全局适配器  端口转发 设置说明

1.选择 VirtualBox-Preferences... 弹出 VirtualBox 窗口; 

2.选择Network 点击创建一个名称为NatNetwork 的网络;

3.选择上面创建的NatNetwork网络，点击右边的一个图标(Edit selected NatNetwork); 

4.选择Port Forwarding;

5.选择IPv4 设置如下转发

|Name|Protocol|HostIP|HostPort|GuestIP|GuestPort|
|----|--------|------|--------|-------|---------|
k8s1-dashboard|TCP|127.0.0.1|9094|10.0.2.4|8009
k8s1-ssh|TCP|127.0.0.1|9091|10.0.2.4|22
k8s2-ssh|TCP|127.0.0.1|9092|10.0.2.6|22
k8s3-ssh|TCP|127.0.0.1|9093|10.0.2.7|22

>  k8s1 ip: 10.0.2.4

>  k8s2 ip: 10.0.2.6

>  k8s3 ip: 10.0.2.7

上述设置完成之后， 还需要在virtualbox  k8s1,k8s2,k8s3 上分别选择 Settings ,

1.选择 Network;

2.选择Adapter1 

3.Attached to: NAT Network  

4.Name:NatNetwork  

> 这样设置之后， 即可以使得k8s1,k8s2,k83 之间相互通信， 也可以使得它们与macos 宿主机通信;

从k8s1 scp文件到 macos 宿主机

    scp -P 9091 root@127.0.0.1:~/f.tar .

从macos 宿主机 scp文件到 k8s1 
    
    scp -P 9091 apt-key.gpg root@127.0.0.1:~/


Heapster 负责k8s集群度量数据采集与容器监控
