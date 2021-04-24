Title: 用vagrant 搭建一个k8s 三节点集群实践过程

> 1master, 2node, k8s-v.1.20.0, fannel network, centos7.9.2009
  
- [0.背景说明](#0背景说明)
- [1.面临问题](#1面临问题)
- [2.解决方案](#2解决方案)
- [3.集群搭建过程](#3集群搭建过程)
  - [3.1 集群模板搭建](#31-集群模板搭建)
    - [3.1.1 初始化虚拟机配置](#311-初始化虚拟机配置)
    - [3.1.2 修改虚拟机配置](#312-修改虚拟机配置)
    - [3.1.3 启动虚拟机](#313-启动虚拟机)
    - [3.1.4 导出为box模板 为后面复用](#314-导出为box模板-为后面复用)
  - [3.2 三节点集群搭建](#32-三节点集群搭建)
    - [3.2.1 集群Vagrantfile 配置文件](#321-集群vagrantfile-配置文件)
    - [3.2.2 验证配置模板](#322-验证配置模板)
    - [3.2.3 安装虚机集群](#323-安装虚机集群)
    - [3.2.4 k8s集群规划](#324-k8s集群规划)
    - [3.2.5 Master节点配置](#325-master节点配置)
      - [3.2.5.1 初始化kubeadm](#3251-初始化kubeadm)
      - [3.2.5.2 配置kubectl](#3252-配置kubectl)
      - [3.2.5.3 部署网络插件flannel](#3253-部署网络插件flannel)
    - [3.2.6 worker节点配置](#326-worker节点配置)
      - [3.2.6.1 添加worker节点](#3261-添加worker节点)
      - [3.2.6.2 kubectl get nodes](#3262-kubectl-get-nodes)
    - [3.2.7 验证 k8s 集群组件](#327-验证-k8s-集群组件)
    - [3.2.8 部署Dashboard](#328-部署dashboard)
    - [3.2.9 解决Dashboard chrome 无法访问问题](#329-解决dashboard-chrome-无法访问问题)
  - [3.3 k8s 集群测试](#33-k8s-集群测试)
## 0.背景说明

- 更新macOS Big Sur 后没法安装 minikube, 也没法通过Helm3 安装kubernetes 集群; 
- 之前安装过的集群因为占用磁盘空间, 没有使用后便删除了;

## 1.面临问题

如何快速在本地搭建一个k8s集群用于实验, 二次开发验证;

## 2.解决方案

step1.	通过vagrant 搭建一个k8s 模板节点 (包含必要的配置和docker, k8s 组件安装)

step2.    基于上述模板快速创建k8s 节点， 通过配置快速完成k8s集群搭建;

## 3.集群搭建过程
### 3.1 集群模板搭建

#### 3.1.1 初始化虚拟机配置

```
vagrant box add http://cloud.centos.org/centos/7/vagrant/x86_64/images/CentOS-7.box --name centos7
vagrant init centos7
```
注意: generic/centos7, 这个vagrant box root账号密码不是vagrant  无法切换到root账号， 不建议使用

#### 3.1.2 修改虚拟机配置

注意: kubernetes v1.21 版本，要求coredns1.8.0， 而阿里源貌似没有coredns1.8.0 , 所以指定安装v1.20.0
```diff
- [ERROR ImagePull]: failed to pull image registry.aliyuncs.com/google_containers/coredns/coredns:v1.8.0: output: Error response from daemon: manifest for registry.aliyuncs.com/google_containers/coredns/coredns:v1.8.0 not found: manifest unknown: manifest unknown
```

```
# vagrantfile 
➜  k8s git:(master) ✗ cat Vagrantfile
# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "centos7"

	config.vm.network "private_network", type:"dhcp"
	config.vm.define "k8s-base"
	config.vm.provider "virtualbox" do |vb|
		vb.name = "k8s-base"
	end
	config.ssh.insert_key = false
	config.vm.hostname = "k8s-base"

	# Vagrant provision shell with root privilege
	# set privileged true
	config.vm.provision "shell", path: "init.sh", privileged: true
end

# init.sh 
➜  k8s git:(master) ✗ cat init.sh
#!/bin/sh

# disable firewalld
iptables -F
systemctl stop firewalld
systemctl disable firewalld

#disable selinux
sudo sed -i 's+SELINUX=enforcing+SELINUX=disabled+' /etc/selinux/config
setenforce 0

# 开启内核模块
modprobe br_netfilter

# 增加网络转发
# 桥接的IPV4流量传递到iptables的链
cat>/etc/sysctl.d/k8s.conf<<EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
EOF

sysctl -p /etc/sysctl.d/k8s.conf

# disable swap
swapoff -a
sed -i 's+/swapfile+#/swapfile+' /etc/fstab
echo vm.swappiness=0 >> /etc/sysctl.conf
sysctl -p

# install k8s components
yum install -y yum-utils device-mapper-persisten-data lvm2 wget vim net-tools

wget -O /etc/yum.repos.d/docker-ce.repo https://download.docker.com/linux/centos/docker-ce.repo

sudo sed -i 's+download.docker.com+mirrors.tuna.tsinghua.edu.cn/docker-ce+' /etc/yum.repos.d/docker-ce.repo

cat>/etc/yum.repos.d/kubernetes.repo<<EOF
[kubernetes]
name=Kubernetes
baseurl=https://mirrors.aliyun.com/kubernetes/yum/repos/kubernetes-el7-x86_64/
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://mirrors.aliyun.com/kubernetes/yum/doc/yum-key.gpg https://mirrors.aliyun.com/kubernetes/yum/doc/rpm-package-key.gpg
EOF

sudo yum makecache fast
yum install -y docker-ce kubelet-1.20.0 kubeadm-1.20.0 kubectl-1.20.0

systemctl start docker
systemctl enable docker

systemctl start kubelet
systemctl enable kubelet

# permit root remote login
sed -i 's+#PermitRootLogin yes+PermitRootLogin yes+' /etc/ssh/sshd_config
sed -i 's+#PermitEmptyPasswords no+PermitEmptyPasswords yes+' /etc/ssh/sshd_config
sed -i 's+PasswordAuthentication no+PasswordAuthentication yes+' /etc/ssh/sshd_config
```

#### 3.1.3 启动虚拟机

```
➜  k8s git:(master) ✗ vagrant up
Bringing machine 'k8s-base' up with 'virtualbox' provider...
==> k8s-base: Importing base box 'centos7'...
==> k8s-base: Matching MAC address for NAT networking...
==> k8s-base: Setting the name of the VM: k8s-base
==> k8s-base: Fixed port collision for 22 => 2222. Now on port 2202.
==> k8s-base: Clearing any previously set network interfaces...
==> k8s-base: Preparing network interfaces based on configuration...
    k8s-base: Adapter 1: nat
    k8s-base: Adapter 2: hostonly
==> k8s-base: Forwarding ports...
    k8s-base: 22 (guest) => 2202 (host) (adapter 1)
==> k8s-base: Booting VM...
==> k8s-base: Waiting for machine to boot. This may take a few minutes...
    k8s-base: SSH address: 127.0.0.1:2202
    k8s-base: SSH username: vagrant
    k8s-base: SSH auth method: private key
==> k8s-base: Machine booted and ready!

    ...

    k8s-base: Complete!
    k8s-base: Created symlink from /etc/systemd/system/multi-user.target.wants/docker.service to /usr/lib/systemd/system/docker.service.
    k8s-base: Created symlink from /etc/systemd/system/multi-user.target.wants/kubelet.service to /usr/lib/systemd/system/kubelet.service.
➜  k8s git:(master) ✗
```

#### 3.1.4 导出为box模板 为后面复用

```
# 关机
➜  k8s git:(master) ✗ vagrant halt
==> k8s-base: Attempting graceful shutdown of VM...
# 导出k8s-v1.20-base.box
➜  k8s git:(master) ✗ vagrant package --base k8s-base --output k8s-v1.20-base.box
==> k8s-base: Clearing any previously set forwarded ports...
==> k8s-base: Exporting VM...
==> k8s-base: Compressing package to: /Users/lihong/workbench/dev/src/github.com/researchlab/dbp/vagrant/base/k8s/k8s-v1.20-base.box
➜  k8s git:(master) ✗ vagrant box list
centos7         (virtualbox, 0)
generic/centos7 (virtualbox, 3.2.16)
k8s-v1.21       (virtualbox, 0)
➜  k8s git:(master) ✗ vagrant box remove k8s-v1.21
Removing box 'k8s-v1.21' (v0) with provider 'virtualbox'...
# 添加box
➜  k8s git:(master) ✗ vagrant box add k8s-v1.20-base.box --name k8s-v1.20
==> box: Box file was not detected as metadata. Adding it directly...
==> box: Adding box 'k8s-v1.20' (v0) for provider:
    box: Unpacking necessary files from: file:///Users/lihong/workbench/dev/src/github.com/researchlab/dbp/vagrant/base/k8s/k8s-v1.20-base.box
==> box: Successfully added box 'k8s-v1.20' (v0) for 'virtualbox'!
```




### 3.2 三节点集群搭建

#### 3.2.1 集群Vagrantfile 配置文件

```
➜  k8s_3node_centos git:(master) ✗ cat Vagrantfile
# -*- mode: ruby -*-
# vi: set ft=ruby :

boxes = [
	{
		:name => "k8s01",
		:eth1 => "192.168.205.10",
		:mem  => "2048",
		:cpu  => "2"
	},
	{
    :name => "k8s02",
		:eth1 => "192.168.205.11",
		:mem  => "1024",
		:cpu  => "1"
	},
	{
    :name => "k8s03",
		:eth1 => "192.168.205.12",
		:mem  => "1024",
		:cpu  => "1"
	}
]

Vagrant.configure("2") do |config|

  config.vm.box = "k8s-v1.20"

	boxes.each do |opts|
		config.vm.define opts[:name] do |config|
			config.vm.hostname = opts[:name]
			config.vm.provider "virtualbox" do |v|
				v.customize ["modifyvm", :id, "--memory", opts[:mem]]
				v.customize ["modifyvm", :id, "--cpus", opts[:cpu]]
			  v.name = opts[:name]
			end
			config.vm.network "private_network", ip: opts[:eth1]
			# 设置公有网络, 如果要设置ip, 则这个ip需要和宿主机在同一个网段，且没有被占用, 这样才能和宿主机通信;
			config.ssh.insert_key = false
		end
	end
end
```

#### 3.2.2 验证配置模板

```
➜  k8s_3node_centos git:(master) ✗ vagrant validate
Vagrantfile validated successfully.
```

#### 3.2.3 安装虚机集群

```
➜  k8s_3node_centos git:(master) ✗ vagrant up
Bringing machine 'k8s01' up with 'virtualbox' provider...
Bringing machine 'k8s02' up with 'virtualbox' provider...
Bringing machine 'k8s03' up with 'virtualbox' provider...
==> k8s01: Importing base box 'k8s-v1.20'...
==> k8s01: Matching MAC address for NAT networking...
==> k8s01: Setting the name of the VM: k8s01
==> k8s01: Fixed port collision for 22 => 2222. Now on port 2200.
==> k8s01: Clearing any previously set network interfaces...
==> k8s01: Preparing network interfaces based on configuration...
    k8s01: Adapter 1: nat
    k8s01: Adapter 2: hostonly
==> k8s01: Forwarding ports...
    k8s01: 22 (guest) => 2200 (host) (adapter 1)
==> k8s01: Running 'pre-boot' VM customizations...
==> k8s01: Booting VM...
==> k8s01: Waiting for machine to boot. This may take a few minutes...
    k8s01: SSH address: 127.0.0.1:2200
    k8s01: SSH username: vagrant
    k8s01: SSH auth method: private key
    k8s01: Warning: Remote connection disconnect. Retrying...
==> k8s01: Machine booted and ready!
[k8s01] GuestAdditions 6.1.18 running --- OK.
==> k8s01: Checking for guest additions in VM...
==> k8s01: Setting hostname...
==> k8s01: Configuring and enabling network interfaces...
==> k8s01: Mounting shared folders...
    k8s01: /vagrant => /Users/lihong/workbench/dev/src/github.com/researchlab/dbp/vagrant/k8s_3node_centos
==> k8s02: Importing base box 'k8s-v1.20'...
==> k8s02: Matching MAC address for NAT networking...
==> k8s02: Setting the name of the VM: k8s02
==> k8s02: Clearing any previously set network interfaces...
==> k8s02: Preparing network interfaces based on configuration...
    k8s02: Adapter 1: nat
    k8s02: Adapter 2: hostonly
==> k8s02: Forwarding ports...
    k8s02: 22 (guest) => 2222 (host) (adapter 1)
==> k8s02: Running 'pre-boot' VM customizations...
==> k8s02: Booting VM...
==> k8s02: Waiting for machine to boot. This may take a few minutes...
    k8s02: SSH address: 127.0.0.1:2222
    k8s02: SSH username: vagrant
    k8s02: SSH auth method: private key
==> k8s02: Machine booted and ready!
[k8s02] GuestAdditions 6.1.18 running --- OK.
==> k8s02: Checking for guest additions in VM...
==> k8s02: Setting hostname...
==> k8s02: Configuring and enabling network interfaces...
==> k8s02: Mounting shared folders...
    k8s02: /vagrant => /Users/lihong/workbench/dev/src/github.com/researchlab/dbp/vagrant/k8s_3node_centos
==> k8s03: Importing base box 'k8s-v1.20'...
==> k8s03: Matching MAC address for NAT networking...
==> k8s03: Setting the name of the VM: k8s03
==> k8s03: Fixed port collision for 22 => 2222. Now on port 2201.
==> k8s03: Clearing any previously set network interfaces...
==> k8s03: Preparing network interfaces based on configuration...
    k8s03: Adapter 1: nat
    k8s03: Adapter 2: hostonly
==> k8s03: Forwarding ports...
    k8s03: 22 (guest) => 2201 (host) (adapter 1)
==> k8s03: Running 'pre-boot' VM customizations...
==> k8s03: Booting VM...
==> k8s03: Waiting for machine to boot. This may take a few minutes...
    k8s03: SSH address: 127.0.0.1:2201
    k8s03: SSH username: vagrant
    k8s03: SSH auth method: private key
==> k8s03: Machine booted and ready!
[k8s03] GuestAdditions 6.1.18 running --- OK.
==> k8s03: Checking for guest additions in VM...
==> k8s03: Setting hostname...
==> k8s03: Configuring and enabling network interfaces...
==> k8s03: Mounting shared folders...
    k8s03: /vagrant => /Users/lihong/workbench/dev/src/github.com/researchlab/dbp/vagrant/k8s_3node_centos
# 安装完成
➜  k8s_3node_centos git:(master) ✗ vagrant status
Current machine states:

k8s01                     running (virtualbox)
k8s02                     running (virtualbox)
k8s03                     running (virtualbox)

This environment represents multiple VMs. The VMs are all listed
above with their current state. For more information about a specific
VM, run `vagrant status NAME`.
➜  k8s_3node_centos git:(master) ✗
```

#### 3.2.4 k8s集群规划

|主机名称|角色|地址(网段根据宿主机定)|
|-------|----|------------------|
|k8s01|master|192.168.205.10
|k8s02|node|192.168.205.11
|k8s03|node|192.168.205.12
#### 3.2.5 Master节点配置

##### 3.2.5.1 初始化kubeadm
```
kubeadm init --apiserver-advertise-address=192.168.205.10 \
             --image-repository registry.aliyuncs.com/google_containers \
             --kubernetes-version v1.20.0 \
             --pod-network-cidr=10.244.0.0/16
```

主节点最低配置 
- [ERROR NumCPU]: the number of available CPUs 1 is less than the required 2
- [ERROR Mem]: the system RAM (990 MB) is less than the minimum 1700 MB

You can also perform this action in beforehand using 'kubeadm config images pull'

注意: 
1. 初始化失败,使用kubeadm reset 进行重置。成功后会生成一串信息,类似kubeadm join --token {token} {master-ip}:6443 --discovery-token-ca-cert-hash sha256:{hash-code} 建议保存。若无法找到该信息,请看下面的操作
2. 需要用root账号执行上面的命令
3. swapoff -a 关闭swap 

- --apiserver-advertise-address 指定api地址,一般为master节点
- --image-repository 指定镜像仓库
- --kubernetes-version 指定k8s版本(截至当前为1.21.0)
- --pod-network-cidr 指定flannel网络(默认不要改)
- --service-cidr：指定service网段,负载均衡ip
- --ignore-preflight-errors=Swap/all：忽略 swap/所有 报错

```
[root@k8s01 vagrant]# kubeadm init --apiserver-advertise-address=192.168.205.10 --image-repository registry.aliyuncs.com/google_containers --kubernetes-version v1.20.0 --pod-network-cidr=10.244.0.0/16
[init] Using Kubernetes version: v1.20.0
[preflight] Running pre-flight checks
	[WARNING IsDockerSystemdCheck]: detected "cgroupfs" as the Docker cgroup driver. The recommended driver is "systemd". Please follow the guide at https://kubernetes.io/docs/setup/cri/
	[WARNING SystemVerification]: this Docker version is not on the list of validated versions: 20.10.6. Latest validated version: 19.03
[preflight] Pulling images required for setting up a Kubernetes cluster
[preflight] This might take a minute or two, depending on the speed of your internet connection
[preflight] You can also perform this action in beforehand using 'kubeadm config images pull'
[certs] Using certificateDir folder "/etc/kubernetes/pki"
[certs] Generating "ca" certificate and key
[certs] Generating "apiserver" certificate and key
[certs] apiserver serving cert is signed for DNS names [k8s01 kubernetes kubernetes.default kubernetes.default.svc kubernetes.default.svc.cluster.local] and IPs [10.96.0.1 192.168.205.10]
[certs] Generating "apiserver-kubelet-client" certificate and key
[certs] Generating "front-proxy-ca" certificate and key
[certs] Generating "front-proxy-client" certificate and key
[certs] Generating "etcd/ca" certificate and key
[certs] Generating "etcd/server" certificate and key
[certs] etcd/server serving cert is signed for DNS names [k8s01 localhost] and IPs [192.168.205.10 127.0.0.1 ::1]
[certs] Generating "etcd/peer" certificate and key
[certs] etcd/peer serving cert is signed for DNS names [k8s01 localhost] and IPs [192.168.205.10 127.0.0.1 ::1]
[certs] Generating "etcd/healthcheck-client" certificate and key
[certs] Generating "apiserver-etcd-client" certificate and key
[certs] Generating "sa" key and public key
[kubeconfig] Using kubeconfig folder "/etc/kubernetes"
[kubeconfig] Writing "admin.conf" kubeconfig file
[kubeconfig] Writing "kubelet.conf" kubeconfig file
[kubeconfig] Writing "controller-manager.conf" kubeconfig file
[kubeconfig] Writing "scheduler.conf" kubeconfig file
[kubelet-start] Writing kubelet environment file with flags to file "/var/lib/kubelet/kubeadm-flags.env"
[kubelet-start] Writing kubelet configuration to file "/var/lib/kubelet/config.yaml"
[kubelet-start] Starting the kubelet
[control-plane] Using manifest folder "/etc/kubernetes/manifests"
[control-plane] Creating static Pod manifest for "kube-apiserver"
[control-plane] Creating static Pod manifest for "kube-controller-manager"
[control-plane] Creating static Pod manifest for "kube-scheduler"
[etcd] Creating static Pod manifest for local etcd in "/etc/kubernetes/manifests"
[wait-control-plane] Waiting for the kubelet to boot up the control plane as static Pods from directory "/etc/kubernetes/manifests". This can take up to 4m0s
[apiclient] All control plane components are healthy after 14.509133 seconds
[upload-config] Storing the configuration used in ConfigMap "kubeadm-config" in the "kube-system" Namespace
[kubelet] Creating a ConfigMap "kubelet-config-1.20" in namespace kube-system with the configuration for the kubelets in the cluster
[upload-certs] Skipping phase. Please see --upload-certs
[mark-control-plane] Marking the node k8s01 as control-plane by adding the labels "node-role.kubernetes.io/master=''" and "node-role.kubernetes.io/control-plane='' (deprecated)"
[mark-control-plane] Marking the node k8s01 as control-plane by adding the taints [node-role.kubernetes.io/master:NoSchedule]
[bootstrap-token] Using token: vuz92q.ic3ny913elk6yj6s
[bootstrap-token] Configuring bootstrap tokens, cluster-info ConfigMap, RBAC Roles
[bootstrap-token] configured RBAC rules to allow Node Bootstrap tokens to get nodes
[bootstrap-token] configured RBAC rules to allow Node Bootstrap tokens to post CSRs in order for nodes to get long term certificate credentials
[bootstrap-token] configured RBAC rules to allow the csrapprover controller automatically approve CSRs from a Node Bootstrap Token
[bootstrap-token] configured RBAC rules to allow certificate rotation for all node client certificates in the cluster
[bootstrap-token] Creating the "cluster-info" ConfigMap in the "kube-public" namespace
[kubelet-finalize] Updating "/etc/kubernetes/kubelet.conf" to point to a rotatable kubelet client certificate and key
[addons] Applied essential addon: CoreDNS
[addons] Applied essential addon: kube-proxy

Your Kubernetes control-plane has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

Alternatively, if you are the root user, you can run:

  export KUBECONFIG=/etc/kubernetes/admin.conf

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
  https://kubernetes.io/docs/concepts/cluster-administration/addons/

Then you can join any number of worker nodes by running the following on each as root:

kubeadm join 192.168.205.10:6443 --token vuz92q.ic3ny913elk6yj6s \
    --discovery-token-ca-cert-hash sha256:0e3da78d00f5c2552c8d62fe92fe2a608489b5b04ead21196cc6f8c11022c647
[root@k8s01 vagrant]#
```
初始化过程说明：

- [preflight] kubeadm 执行初始化前的检查。
- [kubelet-start] 生成kubelet的配置文件”/var/lib/kubelet/config.yaml”
- [certificates] 生成相关的各种token和证书
- [kubeconfig] 生成 KubeConfig 文件，kubelet 需要这个文件与 Master 通信
- [control-plane] 安装 Master 组件，会从指定的 Registry 下载组件的 Docker 镜像。
- [bootstraptoken] 生成token记录下来，后边使用kubeadm join往集群中添加节点时会用到
- [addons] 安装附加组件 kube-proxy 和 kube-dns。 Kubernetes Master 初始化成功，提示如何配置常规用户使用kubectl访问集群。 提示如何安装 Pod 网络。 提示如何注册其他节点到 Cluster。

##### 3.2.5.2 配置kubectl

kubectl 是管理 Kubernetes Cluster 的命令行工具，前面我们已经在所有的节点安装了 kubectl。Master 初始化完成后需要做一些配置工作，然后 kubectl 就能使用了。 依照 kubeadm init 输出的最后提示，推荐用 Linux 普通用户执行 kubectl。
```
#创建普通用户并设置密码123456
useradd centos && echo "centos:123456" | chpasswd centos

#追加sudo权限,并配置sudo免密
sed -i '/^root/a\centos  ALL=(ALL)       NOPASSWD:ALL' /etc/sudoers

#保存集群安全配置文件到当前用户.kube目录
su - centos
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

#启用 kubectl 命令自动补全功能（注销重新登录生效）
echo "source <(kubectl completion bash)" >> ~/.bashrc
```

使用kubectl 
```
# 这个在普通账号下设置，则kubelet 只能在普通账号下使用，如果在root账号下设置则只能在root账号下使用kubectl, 但官方给出的建议是用普通账号配置

mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config


# 查看节点状态可以看到，当前只存在1个master节点，并且这个节点的状态是 NotReady。
$ kubectl get nodes
NAME         STATUS     ROLES    AGE   VERSION
k8s-master   NotReady   master   69m   v1.17.0

# 查看集群状态：确认各个组件都处于healthy状态。
[centos@k8s-master ~]$ kubectl get cs
NAME STATUS MESSAGE ERROR
scheduler Healthy ok
controller-manager Healthy ok
etcd-0 Healthy {"health": "true"}

# 使用 kubectl describe 命令来查看这个节点（Node）对象的详细信息、状态和Conditions
Conditions:
  Type             Status  LastHeartbeatTime                 LastTransitionTime                Reason                       Message
  ----             ------  -----------------                 ------------------                ------                       -------
  MemoryPressure   False   Wed, 22 Jul 2020 17:41:25 +0800   Wed, 22 Jul 2020 16:30:50 +0800   KubeletHasSufficientMemory   kubelet has sufficient memory available
  DiskPressure     False   Wed, 22 Jul 2020 17:41:25 +0800   Wed, 22 Jul 2020 16:30:50 +0800   KubeletHasNoDiskPressure     kubelet has no disk pressure
  PIDPressure      False   Wed, 22 Jul 2020 17:41:25 +0800   Wed, 22 Jul 2020 16:30:50 +0800   KubeletHasSufficientPID      kubelet has sufficient PID available
  Ready            False   Wed, 22 Jul 2020 17:41:25 +0800   Wed, 22 Jul 2020 16:30:50 +0800   KubeletNotReady              runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:docker: network plugin is not ready: cni config uninitialized
```

- 通过 kubectl describe 指令的输出，我们可以看到 NodeNotReady 的原因在于，尚未部署任何网络插件，kube-proxy等组件还处于starting状态。 另外，我们还可以通过 kubectl 检查这个节点上各个系统 Pod 的状态，其中，kube-system 是 Kubernetes 项目预留的系统 Pod 的工作空间（Namepsace，注意它并不是 Linux Namespace，它只是 Kubernetes 划分不同工作空间的单位）

##### 3.2.5.3 部署网络插件flannel

- 要让 Kubernetes Cluster 能够工作，必须安装Pod网络，否则 Pod 之间无法通信。 Kubernetes 支持多种网络方案，这里我们使用 flannel 执行如下命令部署 flannel
- Kubernetes 支持容器网络插件，使用的是一个名叫 CNI 的通用接口，它也是当前容器网络的事实标准，市面上的所有容器网络开源项目都可以通过 CNI 接入 Kubernetes，比如 Flannel、Calico、Canal、Romana 等等，它们的部署方式也都是类似的“一键部署”。

注意: 安装该插件会请求quay.io的镜像, 请确保可以正常访问， 经过测试 quay.mirrors.ustc.edu.cn 可用

注意: 上面的配置kubectl步骤 在普通账号(或root账号)下配置的，则kubectl apply -f kube-fannel.yml 也要在这个普通账号(或者root账号)下执行, 如用的普通账号配置，却用root 账号执行kubectl 命令，会收到如下提示
```diff
- [root@k8s01 vagrant]# kubectl apply -f kube-flannel.yml
- The connection to the server localhost:8080 was refused - did you specify the right host or port?
```

```
# 先下载配置好后部署[建议使用此]
wget https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
## 修改文件中的所有image标签,将 quay.io 改为可用仓库名称 
## 任选[quay.azk8s.cn, quay.mirrors.ustc.edu.cn, quay-mirror.qiniu.com]
## 修改示例:
image: quay.io/coreos/flannel:v0.11.0-amd64 =>
image: quay.azk8s.cn/coreos/flannel:v0.11.0-amd64
# 部署
kubectl apply -f kube-flannel.yml

# 直接部署(可能由于网络原因导致镜像拉取 失败)
kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
## 若你发现部署后coredns的pod无法Running,主谋可能就是flannel节点拉取失败,请使用如下方式
## 镜像地址: quay.io/coreos/flannel 截至[19.11.17]最新版本为0.11.0.*
## 先删除上次部署(删除前最好看看pod事件,查看无法启动原因)
kubectl delete -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
## 拉取镜像(参考底部参考链接,从其他可用仓库拉取镜像)
## 模板: docker pull quay.azk8s.cn/xxx/yyy:zzz
## 其他针对quay.io的可用仓库: quay.mirrors.ustc.edu.cn quay-mirror.qiniu.com 
## 示例操作如下:
docker pull quay.azk8s.cn/coreos/flannel:v0.11.0-amd64
## 根据上一步将缺失的镜像拉取后,修改tag 其实就是将仓库名称改回来
docker tag quay.azk8s.cn/coreos/flannel:v0.11.0-amd64 quay.io/coreos/flannel:v0.11.0-amd64
# 将缺失的镜像全部拉取下来后 再次部署
kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
```

执行过程

```
[vagrant@k8s01 vagrant]$ kubectl apply -f kube-flannel.yml
podsecuritypolicy.policy/psp.flannel.unprivileged created
clusterrole.rbac.authorization.k8s.io/flannel created
clusterrolebinding.rbac.authorization.k8s.io/flannel created
serviceaccount/flannel created
configmap/kube-flannel-cfg created
daemonset.apps/kube-flannel-ds created
[vagrant@k8s01 vagrant]$
```

注意: 没装网络插件时, coredns 的pod 会一直处于Pending状态

装好flannel 网络插件过一会后, coredns pod 的状态会变成Running ,如下表示k8s已经好了

```
[vagrant@k8s01 ~]# kubectl get po -n kube-system
NAME                            READY   STATUS    RESTARTS   AGE
coredns-7f89b7bc75-b4nk5        0/1     Pending   0          22m
coredns-7f89b7bc75-mtl4b        0/1     Pending   0          22m
etcd-k8s01                      1/1     Running   0          22m
kube-apiserver-k8s01            1/1     Running   0          22m
kube-controller-manager-k8s01   1/1     Running   0          22m
kube-flannel-ds-hx6r4           1/1     Running   0          14s
kube-proxy-df2tx                1/1     Running   0          22m
kube-scheduler-k8s01            1/1     Running   0          22m
[vagrant@k8s01 ~]# kubectl get po -n kube-system
NAME                            READY   STATUS    RESTARTS   AGE
coredns-7f89b7bc75-b4nk5        1/1     Running   0          24m
coredns-7f89b7bc75-mtl4b        1/1     Running   0          24m
etcd-k8s01                      1/1     Running   0          24m
kube-apiserver-k8s01            1/1     Running   0          24m
kube-controller-manager-k8s01   1/1     Running   0          24m
kube-flannel-ds-hx6r4           1/1     Running   0          101s
kube-proxy-df2tx                1/1     Running   0          24m
kube-scheduler-k8s01            1/1     Running   0          24m
```

#### 3.2.6 worker节点配置

- Kubernetes 的 Worker 节点跟 Master 节点几乎是相同的，它们运行着的都是一个 kubelet 组件。唯一的区别在于，在 kubeadm init 的过程中，kubelet 启动后，Master 节点上还会自动运行 kube-apiserver、kube-scheduler、kube-controller-manger 这三个系统 Pod。

##### 3.2.6.1 添加worker节点 

添加worker节点, 将主节点配置后生成的kubeadm join 命令复制到worker节点执行即可,

注意: kubeadm join 命令 需要运行在root账号上

```
[root@k8s02 vagrant]# kubeadm join 192.168.205.10:6443 --token vuz92q.ic3ny913elk6yj6s \
>     --discovery-token-ca-cert-hash sha256:0e3da78d00f5c2552c8d62fe92fe2a608489b5b04ead21196cc6f8c11022c647
[preflight] Running pre-flight checks
	[WARNING IsDockerSystemdCheck]: detected "cgroupfs" as the Docker cgroup driver. The recommended driver is "systemd". Please follow the guide at https://kubernetes.io/docs/setup/cri/
	[WARNING SystemVerification]: this Docker version is not on the list of validated versions: 20.10.6. Latest validated version: 19.03
[preflight] Reading configuration from the cluster...
[preflight] FYI: You can look at this config file with 'kubectl -n kube-system get cm kubeadm-config -o yaml'
[kubelet-start] Writing kubelet configuration to file "/var/lib/kubelet/config.yaml"
[kubelet-start] Writing kubelet environment file with flags to file "/var/lib/kubelet/kubeadm-flags.env"
[kubelet-start] Starting the kubelet
[kubelet-start] Waiting for the kubelet to perform the TLS Bootstrap...

This node has joined the cluster:
* Certificate signing request was sent to apiserver and a response was received.
* The Kubelet was informed of the new secure connection details.

Run 'kubectl get nodes' on the control-plane to see this node join the cluster.
```

##### 3.2.6.2 kubectl get nodes 

执行kubectl 命令 收到如下提示， 是因为worker节点没有配置kubectl 

```diff
- [root@k8s02 vagrant]# kubectl get nodes
- The connection to the server localhost:8080 was refused - did you specify the right host or port?
```

worker节点配置kubectl， 是通过拷贝Master节点/etc/kubernetes/admin.conf 文件到自己的相同目录下，并配置环境生效, 具体操作如下,

```
[root@k8s02 vagrant]# scp root@192.168.205.10:/etc/kubernetes/admin.conf /etc/kubernetes/
The authenticity of host '192.168.205.10 (192.168.205.10)' can't be established.
ECDSA key fingerprint is SHA256:cZlJgLcnh9D6GVmeIGuiozQucIaYyh0dypCox8MWQJY.
ECDSA key fingerprint is MD5:30:44:20:c8:52:cb:15:75:b4:15:12:ee:4c:dd:29:b4.
Are you sure you want to continue connecting (yes/no)? yes
Warning: Permanently added '192.168.205.10' (ECDSA) to the list of known hosts.
root@192.168.205.10's password:
admin.conf                                                                  100% 5566     3.8MB/s   00:00
[root@k8s02 vagrant]# echo "export KUBECONFIG=/etc/kubernetes/admin.conf" >> ~/.bash_profile
[root@k8s02 vagrant]# source ~/.bash_profile
[root@k8s02 vagrant]# kubectl get nodes
NAME    STATUS   ROLES                  AGE    VERSION
k8s01   Ready    control-plane,master   19m    v1.20.0
k8s02   Ready    <none>                 3m6s   v1.20.0
[root@k8s02 vagrant]#
```

然后在主节点查看加入的worker节点
```
[root@k8s01 vagrant]# kubectl get nodes
NAME    STATUS   ROLES                  AGE     VERSION
k8s01   Ready    control-plane,master   81m     v1.20.0
k8s02   Ready    <none>                 2m16s   v1.20.0
```

添加worker节点命令说明

```
# 在需要被添加的worker节点(node)上,执行如下一条命令即可
kubeadm join --token {token} {master-ip}:6443 --discovery-token-ca-cert-hash sha256:{hash-code}
# 说明
# token:
#    使用 kubeadm token list 查看可用token列表
#     使用 kubeadm token create 创建一个新的(token过期时使用)
# master-ip
#    填写主节点的ip地址
# hash-code
#    使用以下命令生成:
#    openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //'
```

#### 3.2.7 验证 k8s 集群组件

```
[vagrant@k8s01 ~]$ kubectl get nodes
NAME    STATUS   ROLES                  AGE     VERSION
k8s01   Ready    control-plane,master   24m     v1.20.0
k8s02   Ready    <none>                 8m18s   v1.20.0
k8s03   Ready    <none>                 2m6s    v1.20.0

[vagrant@k8s01 vagrant]$ kubectl get pods --all-namespaces
NAMESPACE              NAME                                         READY   STATUS    RESTARTS   AGE
default                nginx-deployment-585449566-2qnhs             1/1     Running   0          6h13m
default                nginx-deployment-585449566-8j5n9             1/1     Running   0          6h13m
kube-system            coredns-7f89b7bc75-2qlp8                     1/1     Running   0          6h42m
kube-system            coredns-7f89b7bc75-vbsds                     1/1     Running   0          6h42m
kube-system            etcd-k8s01                                   1/1     Running   0          6h42m
kube-system            kube-apiserver-k8s01                         1/1     Running   0          6h42m
kube-system            kube-controller-manager-k8s01                1/1     Running   0          3h25m
kube-system            kube-flannel-ds-7jxz4                        1/1     Running   0          5h50m
kube-system            kube-flannel-ds-hx8xw                        1/1     Running   0          5h50m
kube-system            kube-flannel-ds-kfzzk                        1/1     Running   0          5h50m
kube-system            kube-proxy-6gn5q                             1/1     Running   0          6h26m
kube-system            kube-proxy-dhxlr                             1/1     Running   0          6h20m
kube-system            kube-proxy-wprjf                             1/1     Running   0          6h42m
kube-system            kube-scheduler-k8s01                         1/1     Running   0          3h24m
kubernetes-dashboard   dashboard-metrics-scraper-7445d59dfd-x7hhc   1/1     Running   0          139m
kubernetes-dashboard   kubernetes-dashboard-7d8466d688-p82mj        1/1     Running   0          139m
```
查看所有k8s组件
```
[vagrant@k8s01 vagrant]$ kubectl get cs
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS      MESSAGE                                                                                       ERROR
scheduler            Unhealthy   Get "http://127.0.0.1:10251/healthz": dial tcp 127.0.0.1:10251: connect: connection refused
controller-manager   Unhealthy   Get "http://127.0.0.1:10252/healthz": dial tcp 127.0.0.1:10252: connect: connection refused
etcd-0               Healthy     {"health":"true"}
[vagrant@k8s01 vagrant]$ kubectl get componentstatuses
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS      MESSAGE                                                                                       ERROR
controller-manager   Unhealthy   Get "http://127.0.0.1:10252/healthz": dial tcp 127.0.0.1:10252: connect: connection refused
scheduler            Unhealthy   Get "http://127.0.0.1:10251/healthz": dial tcp 127.0.0.1:10251: connect: connection refused
etcd-0               Healthy     {"health":"true"}
[vagrant@k8s01 vagrant]$
```

出现上面的错误， 是因为在kubernetes1.18.6之后，/etc/kubernetes/manifests下的kube-controller-manager.yaml和kube-scheduler.yaml设置的默认端口是0导致的， 只需要注释掉重启kubelet 即可

kube-controller-manager.yaml文件修改：注释掉27行
```
1 apiVersion: v1
  2 kind: Pod
  3 metadata:
  4   creationTimestamp: null
  5   labels:
  6     component: kube-controller-manager
  7     tier: control-plane
  8   name: kube-controller-manager
  9   namespace: kube-system
 10 spec:
 11   containers:
 12   - command:
 13     - kube-controller-manager
 14     - --allocate-node-cidrs=true
 15     - --authentication-kubeconfig=/etc/kubernetes/controller-manager.conf
 16     - --authorization-kubeconfig=/etc/kubernetes/controller-manager.conf
 17     - --bind-address=127.0.0.1
 18     - --client-ca-file=/etc/kubernetes/pki/ca.crt
 19     - --cluster-cidr=10.244.0.0/16
 20     - --cluster-name=kubernetes
 21     - --cluster-signing-cert-file=/etc/kubernetes/pki/ca.crt
 22     - --cluster-signing-key-file=/etc/kubernetes/pki/ca.key
 23     - --controllers=*,bootstrapsigner,tokencleaner
 24     - --kubeconfig=/etc/kubernetes/controller-manager.conf
 25     - --leader-elect=true
 26     - --node-cidr-mask-size=24
 27   #  - --port=0
 28     - --requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt
 29     - --root-ca-file=/etc/kubernetes/pki/ca.crt
 30     - --service-account-private-key-file=/etc/kubernetes/pki/sa.key
 31     - --service-cluster-ip-range=10.1.0.0/16
 32     - --use-service-account-credentials=true
```
kube-scheduler.yaml配置修改：注释掉19行
```
1 apiVersion: v1
  2 kind: Pod
  3 metadata:
  4   creationTimestamp: null
  5   labels:
  6     component: kube-scheduler
  7     tier: control-plane
  8   name: kube-scheduler
  9   namespace: kube-system
 10 spec:
 11   containers:
 12   - command:
 13     - kube-scheduler
 14     - --authentication-kubeconfig=/etc/kubernetes/scheduler.conf
 15     - --authorization-kubeconfig=/etc/kubernetes/scheduler.conf
 16     - --bind-address=127.0.0.1
 17     - --kubeconfig=/etc/kubernetes/scheduler.conf
 18     - --leader-elect=true
 19   #  - --port=0
```
然后三台机器均重启kubelet
```
[root@k8s01 manifests]# systemctl restart kubelet
```
再次查看，就正常了
```
[vagrant@k8s01 manifests]$ kubectl get cs
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS    MESSAGE             ERROR
scheduler            Healthy   ok
controller-manager   Healthy   ok
etcd-0               Healthy   {"health":"true"}
[vagrant@k8s01 manifests]$
```

#### 3.2.8 部署Dashboard 
1.准备安装kubernetes dashboard的yaml文件
```
wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.0.0-beta8/aio/deploy/recommended.yaml
# 改名
mv recommended.yaml kubernetes-dashboard.yaml
```
2.默认Dashboard只能集群内部访问，修改Service为NodePort类型，并暴露端口到外部
```
vi kubernetes-dashboard.yaml 
kind: Service
apiVersion: v1
metadata:
  labels:
    k8s-app: kubernetes-dashboard
  name: kubernetes-dashboard
  namespace: kubernetes-dashboard
spec:
  type: NodePort
  ports:
    - port: 443
      targetPort: 8443
      nodePort: 30001
  selector:
    k8s-app: kubernetes-dashboard
```
3.创建service account并绑定默认cluster-admin管理员集群角色
```
[vagrant@k8s01 vagrant]$ kubectl create serviceaccount dashboard-admin -n kube-system
serviceaccount/dashboard-admin created
[vagrant@k8s01 vagrant]$ ls
kube-flannel.yml  kubernetes-dashboard.yaml  nginx-deployment.yml  Vagrantfile
[vagrant@k8s01 vagrant]$ kubectl create clusterrolebinding dashboard-admin --clusterrole=cluster-admin --serviceaccount=kube-system:dashboard-admin
clusterrolebinding.rbac.authorization.k8s.io/dashboard-admin created
```
4.查询dashboard token
```
[vagrant@k8s01 vagrant]$ kubectl describe secrets -n kube-system $(kubectl -n kube-system get secret | awk '/dashboard-admin/{print $1}')
Name:         dashboard-admin-token-bt7w2
Namespace:    kube-system
Labels:       <none>
Annotations:  kubernetes.io/service-account.name: dashboard-admin
              kubernetes.io/service-account.uid: e026f261-4330-477e-93e8-508b432fd730

Type:  kubernetes.io/service-account-token

Data
====
ca.crt:     1066 bytes
namespace:  11 bytes
token:      eyJhbGciOiJSUzI1NiIsImtpZCI6ImFlaE1HemFHaDFNSmp3bXljNG9FWTBud3ZrVTFWRGxXVXJSeG14WXI1WG8ifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJrdWJlLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJkYXNoYm9hcmQtYWRtaW4tdG9rZW4tYnQ3dzIiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC5uYW1lIjoiZGFzaGJvYXJkLWFkbWluIiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZXJ2aWNlLWFjY291bnQudWlkIjoiZTAyNmYyNjEtNDMzMC00NzdlLTkzZTgtNTA4YjQzMmZkNzMwIiwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50Omt1YmUtc3lzdGVtOmRhc2hib2FyZC1hZG1pbiJ9.imaBxqrXIOvudszZyJdkdDRQApWsezJ6z0tyotxirsJ6yKnpU67m97NT6E-DWElXSlK1s6bdr7VSynMhp8CAeUqNV0iuUtNybtUkuMME4DeebezmKh60hA0L13FfJ7ayXix9qINFLnXZh_MV_4lqVM2L321Llma57la6mHA5Mzp1b8tlSMI_1XZjCLb0vWqca_6vnWdfY0RSi2KurTQ-8md4SuLo3SGUgxANaMU16TY5kxDgdKQ3aRIgSGh9aQxcCT_YpMf2EMDjeaHocNK68hOVQU01mEH1M5YiLNzV65heltJzK5RzXhG7Ifdkg_qCmO8y-LoMByj5dsJaXQ2LDg
[vagrant@k8s01 vagrant]$
```
5.创建kubernetes-dashboard 
```
创建kubernetes dashboard 
[vagrant@k8s01 vagrant]$ kubectl apply -f kubernetes-dashboard.yaml
namespace/kubernetes-dashboard created
serviceaccount/kubernetes-dashboard created
service/kubernetes-dashboard created
secret/kubernetes-dashboard-certs created
secret/kubernetes-dashboard-csrf created
secret/kubernetes-dashboard-key-holder created
configmap/kubernetes-dashboard-settings created
role.rbac.authorization.k8s.io/kubernetes-dashboard created
clusterrole.rbac.authorization.k8s.io/kubernetes-dashboard created
rolebinding.rbac.authorization.k8s.io/kubernetes-dashboard created
clusterrolebinding.rbac.authorization.k8s.io/kubernetes-dashboard created
deployment.apps/kubernetes-dashboard created
service/dashboard-metrics-scraper created
deployment.apps/dashboard-metrics-scraper created

# 查看kubernetes dashboard
[vagrant@k8s01 vagrant]$ kubectl get pod,svc -n kubernetes-dashboard
NAME                                             READY   STATUS    RESTARTS   AGE
pod/dashboard-metrics-scraper-7445d59dfd-x7hhc   1/1     Running   0          81s
pod/kubernetes-dashboard-7d8466d688-p82mj        1/1     Running   0          81s

NAME                                TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)         AGE
service/dashboard-metrics-scraper   ClusterIP   10.100.185.252   <none>        8000/TCP        81s
service/kubernetes-dashboard        NodePort    10.96.194.52     <none>        443:30001/TCP   81s
```
6.访问dashboard 
```
http://192.168.205.10:30001 
通过前面生成的token访问dashboard 
```

#### 3.2.9 解决Dashboard chrome 无法访问问题

1问题描述
K8S Dashboard安装好以后，通过Firefox浏览器是可以打开的，但通过Google Chrome浏览器，无法成功浏览页面。如图：

2 解决方案
kubeadm自动生成的证书，很多浏览器不支持。所以我们需要自己创建证书。
2.1 创建一个key 目录，存放证书等文件
```
[root@k8s-master ~]# mkdir kubernetes-key
[root@k8s-master ~]# cd kubernetes-key/
```
2.2 生成证书
```
# 1）生成证书请求的key
[root@k8s01 kubernetes-key]# openssl genrsa -out dashboard.key 2048
Generating RSA private key, 2048 bit long modulus
......................................................................................................................................+++
.....+++
e is 65537 (0x10001)

# 2）生成证书请求，下面192.168.205.10为master节点的IP地址
[root@k8s01 kubernetes-key]# openssl req -new -out dashboard.csr -key dashboard.key -subj '/CN=192.168.205.10'

# 3）生成自签证书
[root@k8s01 kubernetes-key]# openssl x509 -req -in dashboard.csr -signkey dashboard.key -out dashboard.crt
Signature ok
subject=/CN=192.168.205.10
Getting Private key
[root@k8s01 kubernetes-key]#
```

3删除原有证书
```
[vagrant@k8s01 opt]$ kubectl delete secret kubernetes-dashboard-certs -n kubernetes-dashboard
secret "kubernetes-dashboard-certs" deleted
```
4 创建新证书的secret
```
[vagrant@k8s01 kubernetes-key]$ ll
total 12
-rw-r--r-- 1 root root  989 4月  23 09:51 dashboard.crt
-rw-r--r-- 1 root root  899 4月  23 09:51 dashboard.csr
-rw-r--r-- 1 root root 1679 4月  23 09:50 dashboard.key
[vagrant@k8s01 kubernetes-key]$ kubectl create secret generic kubernetes-dashboard-certs --from-file=dashboard.key --from-file=dashboard.crt -n kubernetes-dashboard
secret/kubernetes-dashboard-certs created
```
5 删除旧的Pod
```
[vagrant@k8s01 kubernetes-key]$ kubectl get pod -n kubernetes-dashboard
NAME                                         READY   STATUS    RESTARTS   AGE
dashboard-metrics-scraper-7445d59dfd-x7hhc   1/1     Running   0          3h23m
kubernetes-dashboard-7d8466d688-p82mj        1/1     Running   0          3h23m

[vagrant@k8s01 kubernetes-key]$ kubectl delete po dashboard-metrics-scraper-7445d59dfd-x7hhc -n kubernetes-dashboard
pod "dashboard-metrics-scraper-7445d59dfd-x7hhc" deleted
[vagrant@k8s01 kubernetes-key]$ kubectl delete po kubernetes-dashboard-7d8466d688-p82mj -n kubernetes-dashboard
pod "kubernetes-dashboard-7d8466d688-p82mj" deleted

[vagrant@k8s01 kubernetes-key]$ kubectl get po -n kubernetes-dashboard
NAME                                         READY   STATUS    RESTARTS   AGE
dashboard-metrics-scraper-7445d59dfd-9dpj7   1/1     Running   0          33s
kubernetes-dashboard-7d8466d688-xw7p6        1/1     Running   0          16s
```
6 现在就可以正常访问了;

7 除了上述方式，还可以在浏览器通过键盘输入 this is unsafe 也可以访问;

### 3.3 k8s 集群测试

1.创建Deployment
```
# 创建nginx配置文件
vim nginx-deployment.yaml
# 文件内容
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
# 部署
kubectl apply -f nginx-deployment.yaml
```
部署后,我们查看deployment信息
```
# 查看所有节点信息
[vagrant@k8s01 vagrant]$ kubectl get po
NAME                               READY   STATUS    RESTARTS   AGE
nginx-deployment-585449566-2qnhs   1/1     Running   0          3m48s
nginx-deployment-585449566-8j5n9   1/1     Running   0          3m48s
# 查看更详细的节点信息
[vagrant@k8s01 vagrant]$ kubectl get po -o wide
NAME                               READY   STATUS    RESTARTS   AGE     IP           NODE    NOMINATED NODE   READINESS GATES
nginx-deployment-585449566-2qnhs   1/1     Running   0          4m14s   10.244.2.2   k8s03   <none>           <none>
nginx-deployment-585449566-8j5n9   1/1     Running   0          4m14s   10.244.1.2   k8s02   <none>           <none>
[vagrant@k8s01 vagrant]$

#此时，在外部还不能访问nginx 服务
[vagrant@k8s01 vagrant]$ kubectl get svc
NAME         TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
kubernetes   ClusterIP   10.96.0.1    <none>        443/TCP   34m
```
2.暴露资源
发现deployment被分布在了node1和node2上,尝试曝露服务给Service
```
kubectl expose deployment nginx-deployment --port=80 --type=NodePort

[vagrant@k8s01 vagrant]$ kubectl expose deployment nginx-deployment --port=80 --type=NodePort
service/nginx-deployment exposed
[vagrant@k8s01 vagrant]$ kubectl get svc
NAME               TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)        AGE
kubernetes         ClusterIP   10.96.0.1      <none>        443/TCP        36m
nginx-deployment   NodePort    10.109.190.5   <none>        80:30348/TCP   9s
```
查看曝露的服务, 发现对外网曝露的端口是30348，此时应该可以通过localhost:30348 访问页面了， 


如果无法通过localhost:30348 或者浏览器通过192.168.205.10:30348 访问的话，需要修改flannel 的配置并重新部署, 

互Ping一下各个节点的flannel分配的ip,你会发现节点间无法访问,意味着节点间无法通信!什么原因导致的呢?其实是虚拟机限制,我们需要在部署flannel的时候,修改一下flannel启动参数,设置默认网卡即可.看下面操作

第一步， 查看master-IP 对应的网卡
```
[vagrant@k8s01 vagrant]$ ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UP group default qlen 1000
    link/ether 52:54:00:4d:77:d3 brd ff:ff:ff:ff:ff:ff
    inet 10.0.2.15/24 brd 10.0.2.255 scope global noprefixroute dynamic eth0
       valid_lft 83120sec preferred_lft 83120sec
    inet6 fe80::5054:ff:fe4d:77d3/64 scope link
       valid_lft forever preferred_lft forever
3: eth1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UP group default qlen 1000
    link/ether 08:00:27:5c:2a:db brd ff:ff:ff:ff:ff:ff
    inet 192.168.205.10/24 brd 192.168.205.255 scope global noprefixroute eth1
       valid_lft forever preferred_lft forever
    inet6 fe80::a00:27ff:fe5c:2adb/64 scope link
       valid_lft forever preferred_lft forever
4: docker0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue state DOWN group default
    link/ether 02:42:b9:4d:a2:c2 brd ff:ff:ff:ff:ff:ff
    inet 172.17.0.1/16 brd 172.17.255.255 scope global docker0
       valid_lft forever preferred_lft forever
5: flannel.1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UNKNOWN group default
    link/ether 26:b1:37:a2:44:b3 brd ff:ff:ff:ff:ff:ff
    inet 10.244.0.0/32 brd 10.244.0.0 scope global flannel.1
       valid_lft forever preferred_lft forever
    inet6 fe80::24b1:37ff:fea2:44b3/64 scope link
       valid_lft forever preferred_lft forever
6: cni0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UP group default qlen 1000
    link/ether b2:f6:38:37:5e:d1 brd ff:ff:ff:ff:ff:ff
    inet 10.244.0.1/24 brd 10.244.0.255 scope global cni0
       valid_lft forever preferred_lft forever
    inet6 fe80::b0f6:38ff:fe37:5ed1/64 scope link
       valid_lft forever preferred_lft forever
7: veth2000c9b8@if3: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue master cni0 state UP group default
    link/ether 16:94:09:b0:ba:cf brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet6 fe80::1494:9ff:feb0:bacf/64 scope link
       valid_lft forever preferred_lft forever
8: veth357e9cf5@if3: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue master cni0 state UP group default
    link/ether 22:05:bf:c1:3c:23 brd ff:ff:ff:ff:ff:ff link-netnsid 1
    inet6 fe80::2005:bfff:fec1:3c23/64 scope link
       valid_lft forever preferred_lft forever
```
发现192.168.8.110对应的ip分发网卡是 eth1 , 做如下操作
```
# 删除部署的flannel
kubectl delete -f kube-flannel.yml
# 修改kube-flannel.yml文件
# 修改spec.template.spec.containers[x].args 字段
# 一般都是x86架构系(amd64)吧 那就修改第一个DaemonSet 大概在180-200行左右
# 找到对应的平台的DaemonSet
# 在 args 下,添加一行 - -- iface=网卡名
# 上面得到我的网卡 eth1 填写上 - --iface=eth1 即可
# 删除部署的flannel
kubectl delete -f kube-flannel.yml
# 修改kube-flannel.yml文件
# 修改spec.template.spec.containers[x].args 字段
# 一般都是x86架构系(amd64)吧 那就修改第一个DaemonSet 大概在180-200行左右
# 找到对应的平台的DaemonSet
# 在 args 下,添加一行 - -- iface=网卡名
# 上面得到我的网卡 eth1 填写上 - --iface=eth1 即可
containers:
      - name: kube-flannel
        image: quay.io/coreos/flannel:v0.11.0-amd64
        command:
        - /opt/bin/flanneld
        args:
        - --ip-masq
        - --kube-subnet-mgr
        - --iface=eth1
```
保存后, 重新部署
```
kubectl apply -f kube-flannel.yml
```

查看各个节点fannel网段
```
[vagrant@k8s01 vagrant]$ ip a
9: flannel.1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UNKNOWN group default
    link/ether 72:9b:56:d5:d5:b3 brd ff:ff:ff:ff:ff:ff
    inet 10.244.0.0/32 brd 10.244.0.0 scope global flannel.1
       valid_lft forever preferred_lft forever
    inet6 fe80::709b:56ff:fed5:d5b3/64 scope link
       valid_lft forever preferred_lft forever

[vagrant@k8s02 ~]$ ip a
8: flannel.1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UNKNOWN group default
    link/ether ca:43:c1:b7:6a:91 brd ff:ff:ff:ff:ff:ff
    inet 10.244.1.0/32 brd 10.244.1.0 scope global flannel.1
       valid_lft forever preferred_lft forever
    inet6 fe80::c843:c1ff:feb7:6a91/64 scope link
       valid_lft forever preferred_lft forever

[vagrant@k8s03 ~]$ ip a
8: flannel.1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UNKNOWN group default
    link/ether ee:0d:a4:e6:7a:a4 brd ff:ff:ff:ff:ff:ff
    inet 10.244.2.0/32 brd 10.244.2.0 scope global flannel.1
       valid_lft forever preferred_lft forever
    inet6 fe80::ec0d:a4ff:fee6:7aa4/64 scope link
       valid_lft forever preferred_lft forever
[vagrant@k8s03 ~]$

节点网络互通验证
[vagrant@k8s01 ~]$ kubectl get pods -o wide
NAME                               READY   STATUS    RESTARTS   AGE   IP           NODE    NOMINATED NODE   READINESS GATES
nginx-deployment-585449566-2qnhs   1/1     Running   0          27m   10.244.2.2   k8s03   <none>           <none>
nginx-deployment-585449566-8j5n9   1/1     Running   0          27m   10.244.1.2   k8s02   <none>           <none>
[vagrant@k8s01 ~]$ ping 10.244.1.2
PING 10.244.1.2 (10.244.1.2) 56(84) bytes of data.
64 bytes from 10.244.1.2: icmp_seq=1 ttl=63 time=0.717 ms

[vagrant@k8s01 ~]$ ping 10.244.2.2 -c 2
PING 10.244.2.2 (10.244.2.2) 56(84) bytes of data.
64 bytes from 10.244.2.2: icmp_seq=1 ttl=63 time=0.819 ms
64 bytes from 10.244.2.2: icmp_seq=2 ttl=63 time=0.715 ms

--- 10.244.2.2 ping statistics ---
2 packets transmitted, 2 received, 0% packet loss, time 1000ms
rtt min/avg/max/mdev = 0.715/0.767/0.819/0.052 ms
```

再次访问nginx 页面
```
[vagrant@k8s01 ~]$ kubectl get svc
NAME               TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)        AGE
kubernetes         ClusterIP   10.96.0.1      <none>        443/TCP        58m
nginx-deployment   NodePort    10.109.190.5   <none>        80:30348/TCP   22m
# 外网访问第一个节点
[vagrant@k8s01 ~]$ curl -I 192.168.205.10:30348
HTTP/1.1 200 OK
Server: nginx/1.19.10
Date: Fri, 23 Apr 2021 03:09:34 GMT
Content-Type: text/html
Content-Length: 612
Last-Modified: Tue, 13 Apr 2021 15:13:59 GMT
Connection: keep-alive
ETag: "6075b537-264"
Accept-Ranges: bytes

# 外网访问第二个节点
[vagrant@k8s01 ~]$ curl -I 192.168.205.11:30348
HTTP/1.1 200 OK
Server: nginx/1.19.10
Date: Fri, 23 Apr 2021 03:09:44 GMT
Content-Type: text/html
Content-Length: 612
Last-Modified: Tue, 13 Apr 2021 15:13:59 GMT
Connection: keep-alive
ETag: "6075b537-264"
Accept-Ranges: bytes
```
