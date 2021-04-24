Title: kubeadm部署k8s(v1.20.0)三节点集群

> 1master, 2node, k8s-v.1.20.0, calico network, k8s-dashboard

> centos7.9.2009(on virtualbox by vagrant)
  
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
      - [3.2.5.3 部署网络插件calico](#3253-部署网络插件calico)
    - [3.2.6 worker节点配置](#326-worker节点配置)
      - [3.2.6.1 添加worker节点](#3261-添加worker节点)
      - [3.2.6.2 kubectl get nodes](#3262-kubectl-get-nodes)
    - [3.2.7 验证 k8s 集群组件](#327-验证-k8s-集群组件)
    - [3.2.8 kube-proxy开启ipvs](#328-kube-proxy开启ipvs)
    - [3.2.9 部署Dashboard](#329-部署dashboard)
  - [3.3 k8s 集群测试](#33-k8s-集群测试)

## 1.面临问题

如何快速在本地搭建一个使用calico的k8s集群用于实验, 二次开发验证;

## 2.解决方案

step1. 通过vagrant 搭建一个k8s 模板节点 (包含必要的配置和docker, k8s 组件安装)

step2. 基于上述模板快速创建k8s 节点， 通过配置快速完成k8s集群搭建;

## 3.集群搭建过程

- 集群模板配置及脚本文件: https://github.com/researchlab/dbp/tree/master/vagrant/base/k8s

- 集群配置及脚本文件: https://github.com/researchlab/dbp/tree/master/vagrant/k8s_3node_centos_calico
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

# 同步时间
# 方案一
# yum install ntpdate -y
# ntpdate http://cn.pool.ntp.org

#方案二
#yum install -y chrony
#systemctl enable --now chronyd
#chronyc sources && timedatectl

# 增加网络转发
# 桥接的IPV4流量传递到iptables 的链
cat>/etc/sysctl.d/k8s.conf<<EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
EOF

sysctl -p /etc/sysctl.d/k8s.conf

# kube-proxy开启ipvs
# 参考：https://github.com/kubernetes/kubernetes/tree/master/pkg/proxy/ipvs
# kuber-proxy代理支持iptables和ipvs两种模式，使用ipvs模式需要在初始化集群前加载要求的ipvs模块并安装ipset工具。另外，针对Linux kernel 4.19以上的内核版本使用nf_conntrack 代替nf_conntrack_ipv4。
# 方案一
# cat>/etc/sysconfig/modules/ipvs.modules<<EOF
# #!/bin/bash
# # Load IPVS at boot
# modprobe -- ip_vs
# modprobe -- ip_vs_rr
# modprobe -- ip_vs_wrr
# modprobe -- ip_vs_sh
# modprobe -- nf_conntrack_ipv4
# EOF

# 方案二
cat>/etc/modules-load.d/ipvs.conf<<EOF
# Load IPVS at boot
ip_vs
ip_vs_rr
ip_vs_wrr
ip_vs_sh
nf_conntrack_ipv4
EOF

systemctl enable --now systemd-modules-load.service

#确认内核模块加载成功
lsmod |grep -e ip_vs -e nf_conntrack_ipv4

#安装ipset、ipvsadm
yum install -y ipset ipvsadm


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
➜  k8s git:(master) ✗ vagrant box add k8s-v1.20-base.box --name k8s-v1.20.1
==> box: Box file was not detected as metadata. Adding it directly...
==> box: Adding box 'k8s-v1.20.1' (v0) for provider:
    box: Unpacking necessary files from: file:///Users/lihong/workbench/dev/src/github.com/researchlab/dbp/vagrant/base/k8s/k8s-v1.20-base.box
==> box: Successfully added box 'k8s-v1.20.1' (v0) for 'virtualbox'!
```

### 3.2 三节点集群搭建

#### 3.2.1 集群Vagrantfile 配置文件

```
➜  k8s_3node_centos_calico git:(master) ✗ cat Vagrantfile
# -*- mode: ruby -*-
# vi: set ft=ruby :

boxes = [
	{
		:name => "k8s-calico-01",
		:eth1 => "192.168.206.10",
		:mem  => "2048",
		:cpu  => "2"
	},
	{
    :name => "k8s-calico-02",
		:eth1 => "192.168.206.11",
		:mem  => "1024",
		:cpu  => "1"
	},
	{
    :name => "k8s-calico-03",
		:eth1 => "192.168.206.12",
		:mem  => "1024",
		:cpu  => "1"
	}
]

Vagrant.configure("2") do |config|

	config.vm.box = "k8s-v1.20.1"

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
			# 如果不清楚, 也可以不设置, 在创建虚拟机时会自动分配一个和宿主机同网段可用的ip
			#config.vm.network "public_network"
			# base box need set insert_key = false
			config.ssh.insert_key = false
			config.vm.provision "shell", path: "init.sh", privileged: true
		end
	end
end
➜  k8s_3node_centos_calico git:(master) ✗ cat init.sh
#!/bin/sh

#配置hosts解析
cat>/etc/hosts<<EOF
192.168.206.10 k8s-calico-01
192.168.206.11 k8s-calico-02
192.168.206.12 k8s-calico-03
EOF
```

#### 3.2.2 验证配置模板

```
➜  k8s_3node_centos git:(master) ✗ vagrant validate
Vagrantfile validated successfully.
```

#### 3.2.3 安装虚机集群

```
➜  k8s_3node_centos_calico git:(master) ✗ vagrant up
Bringing machine 'k8s-calico-01' up with 'virtualbox' provider...
Bringing machine 'k8s-calico-02' up with 'virtualbox' provider...
Bringing machine 'k8s-calico-03' up with 'virtualbox' provider...
==> k8s-calico-01: Importing base box 'k8s-v1.20.1'...
==> k8s-calico-01: Matching MAC address for NAT networking...
==> k8s-calico-01: Setting the name of the VM: k8s-calico-01
==> k8s-calico-01: Fixed port collision for 22 => 2222. Now on port 2202.
==> k8s-calico-01: Clearing any previously set network interfaces...
==> k8s-calico-01: Preparing network interfaces based on configuration...
    k8s-calico-01: Adapter 1: nat
    k8s-calico-01: Adapter 2: hostonly
==> k8s-calico-01: Forwarding ports...
    k8s-calico-01: 22 (guest) => 2202 (host) (adapter 1)
==> k8s-calico-01: Running 'pre-boot' VM customizations...
==> k8s-calico-01: Booting VM...
==> k8s-calico-01: Waiting for machine to boot. This may take a few minutes...
    k8s-calico-01: SSH address: 127.0.0.1:2202
    k8s-calico-01: SSH username: vagrant
    k8s-calico-01: SSH auth method: private key
==> k8s-calico-01: Machine booted and ready!
[k8s-calico-01] GuestAdditions 6.1.18 running --- OK.
==> k8s-calico-01: Checking for guest additions in VM...
==> k8s-calico-01: Setting hostname...
==> k8s-calico-01: Configuring and enabling network interfaces...
==> k8s-calico-01: Mounting shared folders...
    k8s-calico-01: /vagrant => /Users/lihong/workbench/dev/src/github.com/researchlab/dbp/vagrant/k8s_3node_centos_calico
==> k8s-calico-01: Running provisioner: shell...
    k8s-calico-01: Running: /var/folders/5q/7gy5r2h91_s477x2g8vn30hc0000gn/T/vagrant-shell20210424-10942-ri8b2f.sh
==> k8s-calico-02: Importing base box 'k8s-v1.20.1'...
==> k8s-calico-02: Matching MAC address for NAT networking...
==> k8s-calico-02: Setting the name of the VM: k8s-calico-02
==> k8s-calico-02: Fixed port collision for 22 => 2222. Now on port 2203.
==> k8s-calico-02: Clearing any previously set network interfaces...
==> k8s-calico-02: Preparing network interfaces based on configuration...
    k8s-calico-02: Adapter 1: nat
    k8s-calico-02: Adapter 2: hostonly
==> k8s-calico-02: Forwarding ports...
    k8s-calico-02: 22 (guest) => 2203 (host) (adapter 1)
==> k8s-calico-02: Running 'pre-boot' VM customizations...
==> k8s-calico-02: Booting VM...
==> k8s-calico-02: Waiting for machine to boot. This may take a few minutes...
    k8s-calico-02: SSH address: 127.0.0.1:2203
    k8s-calico-02: SSH username: vagrant
    k8s-calico-02: SSH auth method: private key
==> k8s-calico-02: Machine booted and ready!
[k8s-calico-02] GuestAdditions 6.1.18 running --- OK.
==> k8s-calico-02: Checking for guest additions in VM...
==> k8s-calico-02: Setting hostname...
==> k8s-calico-02: Configuring and enabling network interfaces...
==> k8s-calico-02: Mounting shared folders...
    k8s-calico-02: /vagrant => /Users/lihong/workbench/dev/src/github.com/researchlab/dbp/vagrant/k8s_3node_centos_calico
==> k8s-calico-02: Running provisioner: shell...
    k8s-calico-02: Running: /var/folders/5q/7gy5r2h91_s477x2g8vn30hc0000gn/T/vagrant-shell20210424-10942-e51jmj.sh
==> k8s-calico-03: Importing base box 'k8s-v1.20.1'...
==> k8s-calico-03: Matching MAC address for NAT networking...
==> k8s-calico-03: Setting the name of the VM: k8s-calico-03
==> k8s-calico-03: Fixed port collision for 22 => 2222. Now on port 2204.
==> k8s-calico-03: Clearing any previously set network interfaces...
==> k8s-calico-03: Preparing network interfaces based on configuration...
    k8s-calico-03: Adapter 1: nat
    k8s-calico-03: Adapter 2: hostonly
==> k8s-calico-03: Forwarding ports...
    k8s-calico-03: 22 (guest) => 2204 (host) (adapter 1)
==> k8s-calico-03: Running 'pre-boot' VM customizations...
==> k8s-calico-03: Booting VM...
==> k8s-calico-03: Waiting for machine to boot. This may take a few minutes...
    k8s-calico-03: SSH address: 127.0.0.1:2204
    k8s-calico-03: SSH username: vagrant
    k8s-calico-03: SSH auth method: private key
==> k8s-calico-03: Machine booted and ready!
[k8s-calico-03] GuestAdditions 6.1.18 running --- OK.
==> k8s-calico-03: Checking for guest additions in VM...
==> k8s-calico-03: Setting hostname...
==> k8s-calico-03: Configuring and enabling network interfaces...
==> k8s-calico-03: Mounting shared folders...
    k8s-calico-03: /vagrant => /Users/lihong/workbench/dev/src/github.com/researchlab/dbp/vagrant/k8s_3node_centos_calico
==> k8s-calico-03: Running provisioner: shell...
    k8s-calico-03: Running: /var/folders/5q/7gy5r2h91_s477x2g8vn30hc0000gn/T/vagrant-shell20210424-10942-x53vju.sh
➜  k8s_3node_centos_calico git:(master) ✗ vagrant status
Current machine states:

k8s-calico-01             running (virtualbox)
k8s-calico-02             running (virtualbox)
k8s-calico-03             running (virtualbox)

This environment represents multiple VMs. The VMs are all listed
above with their current state. For more information about a specific
VM, run `vagrant status NAME`.
➜  k8s_3node_centos_calico git:(master) ✗ vagrant ssh k8s-calico-01
[vagrant@k8s-calico-01 ~]$ cat /etc/hosts
192.168.206.10 k8s-calico-01
192.168.206.11 k8s-calico-02
192.168.206.12 k8s-calico-03
[vagrant@k8s-calico-01 ~]$
```

#### 3.2.4 k8s集群规划

|主机名称|角色|地址(网段根据宿主机定)|
|-------|----|------------------|
|k8s-calico-01|master|192.168.206.10
|k8s-calico-02|node|192.168.206.11
|k8s-calico-03|node|192.168.206.12
#### 3.2.5 Master节点配置

##### 3.2.5.1 初始化kubeadm
```
kubeadm init --apiserver-advertise-address=192.168.206.10 \
             --image-repository registry.aliyuncs.com/google_containers \
             --kubernetes-version v1.20.0 \
             --pod-network-cidr=192.168.0.0/16
```

主节点最低配置 
- [ERROR NumCPU]: the number of available CPUs 1 is less than the required 2
- [ERROR Mem]: the system RAM (990 MB) is less than the minimum 1700 MB

You can also perform this action in beforehand using 'kubeadm config images pull'

注意: 
1. 初始化失败,使用kubeadm reset 进行重置。成功后会生成一串信息,类似kubeadm join --token {token} {master-ip}:6443 --discovery-token-ca-cert-hash sha256:{hash-code} 建议保存。若无法找到该信息,请看下面的操作
2. 需要用root账号执行上面的命令
3. swapoff -a 关闭swap 

- --apiserver-advertise-address(可选) 指定api地址,kubeadm 会使用默认网关所在的网络接口广播其主节点的 IP 地址。若需使用其他网络接口，请给 kubeadm init 设置 --apiserver-advertise-address= 参数, 一般为master节点IP地址。
- --image-repository 指定镜像仓库, Kubenetes默认Registries地址是k8s.gcr.io，国内无法访问，在1.13版本后可以增加–image-repository参数，将其指定为可访问的镜像地址，这里使用registry.aliyuncs.com/google_containers。
- --kubernetes-version 指定k8s版本(截至当前为1.21.0)
- --pod-network-cidr 选择一个 Pod 网络插件，并检查是否在 kubeadm 初始化过程中需要传入什么参数。这个取决于您选择的网络插件，您可能需要设置 --Pod-network-cidr 来指定网络驱动的CIDR。Kubernetes 支持多种网络方案，而且不同网络方案对 --pod-network-cidr有自己的要求，flannel设置为 10.244.0.0/16，calico设置为192.168.0.0/16
- --service-cidr：指定service网段,负载均衡ip
- --ignore-preflight-errors=Swap/all：忽略 swap/所有 报错

```
[root@k8s-calico-01 vagrant]# kubeadm init --apiserver-advertise-address=192.168.206.10 \
>              --image-repository registry.aliyuncs.com/google_containers \
>              --kubernetes-version v1.20.0 \
>              --pod-network-cidr=192.168.0.0/16
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
[certs] apiserver serving cert is signed for DNS names [k8s-calico-01 kubernetes kubernetes.default kubernetes.default.svc kubernetes.default.svc.cluster.local] and IPs [10.96.0.1 192.168.206.10]
[certs] Generating "apiserver-kubelet-client" certificate and key
[certs] Generating "front-proxy-ca" certificate and key
[certs] Generating "front-proxy-client" certificate and key
[certs] Generating "etcd/ca" certificate and key
[certs] Generating "etcd/server" certificate and key
[certs] etcd/server serving cert is signed for DNS names [k8s-calico-01 localhost] and IPs [192.168.206.10 127.0.0.1 ::1]
[certs] Generating "etcd/peer" certificate and key
[certs] etcd/peer serving cert is signed for DNS names [k8s-calico-01 localhost] and IPs [192.168.206.10 127.0.0.1 ::1]
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
[apiclient] All control plane components are healthy after 16.004730 seconds
[upload-config] Storing the configuration used in ConfigMap "kubeadm-config" in the "kube-system" Namespace
[kubelet] Creating a ConfigMap "kubelet-config-1.20" in namespace kube-system with the configuration for the kubelets in the cluster
[upload-certs] Skipping phase. Please see --upload-certs
[mark-control-plane] Marking the node k8s-calico-01 as control-plane by adding the labels "node-role.kubernetes.io/master=''" and "node-role.kubernetes.io/control-plane='' (deprecated)"
[mark-control-plane] Marking the node k8s-calico-01 as control-plane by adding the taints [node-role.kubernetes.io/master:NoSchedule]
[bootstrap-token] Using token: 2t6gy6.oquh2cm42kwn0k2q
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

kubeadm join 192.168.206.10:6443 --token 2t6gy6.oquh2cm42kwn0k2q \
    --discovery-token-ca-cert-hash sha256:522e1a7708584995dd6dff705bf4d9612a1ff463735b4d58631c77b2c04b5521
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

[vagrant@k8s-calico-01 ~]$ mkdir -p $HOME/.kube
[vagrant@k8s-calico-01 ~]$ sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
[vagrant@k8s-calico-01 ~]$ sudo chown $(id -u):$(id -g) $HOME/.kube/config

# 查看节点状态可以看到，当前只存在1个master节点，并且这个节点的状态是 NotReady。
[vagrant@k8s-calico-01 ~]$ kubectl get nodes
NAME            STATUS     ROLES                  AGE     VERSION
k8s-calico-01   NotReady   control-plane,master   6m42s   v1.20.0
```


查看集群状态：确认各个组件都处于healthy状态。
```
[vagrant@k8s-calico-01 ~]$ kubectl get cs
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS      MESSAGE                                                                                       ERROR
scheduler            Unhealthy   Get "http://127.0.0.1:10251/healthz": dial tcp 127.0.0.1:10251: connect: connection refused
controller-manager   Unhealthy   Get "http://127.0.0.1:10252/healthz": dial tcp 127.0.0.1:10252: connect: connection refused
etcd-0               Healthy     {"health":"true"}
```
出现上面的错误， 是因为在kubernetes1.18.6之后，/etc/kubernetes/manifests下的kube-controller-manager.yaml和kube-scheduler.yaml设置的默认端口是0导致的， 只需要注释掉重启kubelet 即可。

```
# load as root 
systemctl restart kubelet 

# load as vagrant 
[vagrant@k8s-calico-01 ~]$ kubectl get cs
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS    MESSAGE             ERROR
scheduler            Healthy   ok
controller-manager   Healthy   ok
etcd-0               Healthy   {"health":"true"}
```

查看节点状态
```
# 使用 kubectl describe 命令来查看这个节点（Node）对象的详细信息、状态和Conditions
[vagrant@k8s-calico-01 ~]$ kubectl describe node k8s-calico-01
Name:               k8s-calico-01
Roles:              control-plane,master
Labels:             beta.kubernetes.io/arch=amd64
                    beta.kubernetes.io/os=linux
                    kubernetes.io/arch=amd64
                    kubernetes.io/hostname=k8s-calico-01
                    kubernetes.io/os=linux
                    node-role.kubernetes.io/control-plane=
                    node-role.kubernetes.io/master=
Annotations:        kubeadm.alpha.kubernetes.io/cri-socket: /var/run/dockershim.sock
                    node.alpha.kubernetes.io/ttl: 0
                    volumes.kubernetes.io/controller-managed-attach-detach: true
CreationTimestamp:  Sat, 24 Apr 2021 10:09:21 +0000
Taints:             node-role.kubernetes.io/master:NoSchedule
                    node.kubernetes.io/not-ready:NoSchedule
Unschedulable:      false
Lease:
  HolderIdentity:  k8s-calico-01
  AcquireTime:     <unset>
  RenewTime:       Sat, 24 Apr 2021 10:19:01 +0000
Conditions:
  Type             Status  LastHeartbeatTime                 LastTransitionTime                Reason                       Message
  ----             ------  -----------------                 ------------------                ------                       -------
  MemoryPressure   False   Sat, 24 Apr 2021 10:14:42 +0000   Sat, 24 Apr 2021 10:09:16 +0000   KubeletHasSufficientMemory   kubelet has sufficient memory available
  DiskPressure     False   Sat, 24 Apr 2021 10:14:42 +0000   Sat, 24 Apr 2021 10:09:16 +0000   KubeletHasNoDiskPressure     kubelet has no disk pressure
  PIDPressure      False   Sat, 24 Apr 2021 10:14:42 +0000   Sat, 24 Apr 2021 10:09:16 +0000   KubeletHasSufficientPID      kubelet has sufficient PID available
  Ready            False   Sat, 24 Apr 2021 10:14:42 +0000   Sat, 24 Apr 2021 10:09:16 +0000   KubeletNotReady              runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:docker: network plugin is not ready: cni config uninitialized
```

- 通过 kubectl describe 指令的输出，我们可以看到 NodeNotReady 的原因在于，尚未部署任何网络插件，kube-proxy等组件还处于starting状态。 另外，我们还可以通过 kubectl 检查这个节点上各个系统 Pod 的状态，其中，kube-system 是 Kubernetes 项目预留的系统 Pod 的工作空间（Namepsace，注意它并不是 Linux Namespace，它只是 Kubernetes 划分不同工作空间的单位）

##### 3.2.5.3 部署网络插件calico

- 要让 Kubernetes Cluster 能够工作，必须安装Pod网络，否则 Pod 之间无法通信。 Kubernetes 支持多种网络方案，这里我们使用 calico 执行如下命令部署 flannel
- 官方文档参考: https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/#pod-network
- 官方文档参考: https://docs.projectcalico.org/v3.10/getting-started/kubernetes/
- 为使calico正常工作，你需要传递–pod-network-cidr=192.168.0.0/16到kubeadm init或更新calico.yml文件，以与您的pod网络相匹配

```
# 先下载配置好后部署[建议使用此]
wget wget https://docs.projectcalico.org/v3.10/manifests/calico.yaml

# 部署
kubectl apply -f calico.yaml
```

执行过程

```
[vagrant@k8s-calico-01 vagrant]$ kubectl apply -f calico.yaml
configmap/calico-config created
Warning: apiextensions.k8s.io/v1beta1 CustomResourceDefinition is deprecated in v1.16+, unavailable in v1.22+; use apiextensions.k8s.io/v1 CustomResourceDefinition
customresourcedefinition.apiextensions.k8s.io/felixconfigurations.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/ipamblocks.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/blockaffinities.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/ipamhandles.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/ipamconfigs.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/bgppeers.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/bgpconfigurations.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/ippools.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/hostendpoints.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/clusterinformations.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/globalnetworkpolicies.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/globalnetworksets.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/networkpolicies.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/networksets.crd.projectcalico.org created
clusterrole.rbac.authorization.k8s.io/calico-kube-controllers created
clusterrolebinding.rbac.authorization.k8s.io/calico-kube-controllers created
clusterrole.rbac.authorization.k8s.io/calico-node created
clusterrolebinding.rbac.authorization.k8s.io/calico-node created
daemonset.apps/calico-node created
serviceaccount/calico-node created
deployment.apps/calico-kube-controllers created
serviceaccount/calico-kube-controllers created
```

注意: 没装网络插件时, coredns 的pod 会一直处于Pending状态

装好calico网络插件过一会后, coredns pod 的状态会变成Running ,如下表示k8s已经好了

```
[vagrant@k8s-calico-01 vagrant]$ kubectl get po -n kube-system
NAME                                       READY   STATUS     RESTARTS   AGE
calico-kube-controllers-7854b85cf7-nh5b9   0/1     Pending    0          24s
calico-node-flkr5                          0/1     Init:0/3   0          24s
coredns-7f89b7bc75-fc4dd                   0/1     Pending    0          19m
coredns-7f89b7bc75-sgkms                   0/1     Pending    0          19m
etcd-k8s-calico-01                         1/1     Running    0          19m
kube-apiserver-k8s-calico-01               1/1     Running    0          19m
kube-controller-manager-k8s-calico-01      1/1     Running    0          6m9s
kube-proxy-rxx9g                           1/1     Running    0          19m
kube-scheduler-k8s-calico-01               1/1     Running    0          5m57s
[vagrant@k8s-calico-01 vagrant]$ kubectl get po -n kube-system
NAME                                       READY   STATUS              RESTARTS   AGE
calico-kube-controllers-7854b85cf7-nh5b9   0/1     ContainerCreating   0          115s
calico-node-flkr5                          0/1     Running             0          115s
coredns-7f89b7bc75-fc4dd                   1/1     Running             0          21m
coredns-7f89b7bc75-sgkms                   1/1     Running             0          21m
etcd-k8s-calico-01                         1/1     Running             0          21m
kube-apiserver-k8s-calico-01               1/1     Running             0          21m
kube-controller-manager-k8s-calico-01      1/1     Running             0          7m40s
kube-proxy-rxx9g                           1/1     Running             0          21m
kube-scheduler-k8s-calico-01               1/1     Running             0          7m28s
[vagrant@k8s-calico-01 vagrant]$ kubectl get po -n kube-system
NAME                                       READY   STATUS    RESTARTS   AGE
calico-kube-controllers-7854b85cf7-nh5b9   1/1     Running   0          2m47s
calico-node-flkr5                          0/1     Running   1          2m47s
coredns-7f89b7bc75-fc4dd                   1/1     Running   0          22m
coredns-7f89b7bc75-sgkms                   1/1     Running   0          22m
etcd-k8s-calico-01                         1/1     Running   0          22m
kube-apiserver-k8s-calico-01               1/1     Running   0          22m
kube-controller-manager-k8s-calico-01      1/1     Running   0          8m32s
kube-proxy-rxx9g                           1/1     Running   0          22m
kube-scheduler-k8s-calico-01               1/1     Running   0          8m20s
```

#### 3.2.6 worker节点配置

- Kubernetes 的 Worker 节点跟 Master 节点几乎是相同的，它们运行着的都是一个 kubelet 组件。唯一的区别在于，在 kubeadm init 的过程中，kubelet 启动后，Master 节点上还会自动运行 kube-apiserver、kube-scheduler、kube-controller-manger 这三个系统 Pod。

##### 3.2.6.1 添加worker节点 

添加worker节点, 将主节点配置后生成的kubeadm join 命令复制到worker节点执行即可,

注意: kubeadm join 命令 需要运行在root账号上

```
[vagrant@k8s-calico-02 ~]$ su
Password:
[root@k8s-calico-02 vagrant]# kubeadm join 192.168.206.10:6443 --token 2t6gy6.oquh2cm42kwn0k2q \
>     --discovery-token-ca-cert-hash sha256:522e1a7708584995dd6dff705bf4d9612a1ff463735b4d58631c77b2c04b5521
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

注意: worker节点 配置 kubectl 最好在root账号下配置， 因为要在/etc/kubernetes/写admin.conf 文件; 

```
[root@k8s-calico-02 vagrant]# scp root@192.168.206.10:/etc/kubernetes/admin.conf /etc/kubernetes/
The authenticity of host '192.168.206.10 (192.168.206.10)' can't be established.
ECDSA key fingerprint is SHA256:LUXAd/+kFyDJFh9ykJRu/6MkCili87ubr76EMH6YJug.
ECDSA key fingerprint is MD5:76:d5:b7:00:41:ea:b3:96:6e:f4:1f:66:c2:f4:f3:ba.
Are you sure you want to continue connecting (yes/no)? yes
Warning: Permanently added '192.168.206.10' (ECDSA) to the list of known hosts.
root@192.168.206.10's password:
admin.conf                                                                                                   100% 5570     2.9MB/s   00:00
[root@k8s-calico-02 vagrant]# echo "export KUBECONFIG=/etc/kubernetes/admin.conf" >> ~/.bash_profile
[root@k8s-calico-02 vagrant]# source ~/.bash_profile
[root@k8s-calico-02 vagrant]# kubectl get nodes
NAME            STATUS   ROLES                  AGE     VERSION
k8s-calico-01   Ready    control-plane,master   31m     v1.20.0
k8s-calico-02   Ready    <none>                 5m19s   v1.20.0
[root@k8s-calico-02 vagrant]#
```

#### 3.2.7 验证 k8s 集群组件

```
[vagrant@k8s-calico-01 ~]$ kubectl get nodes
NAME            STATUS   ROLES                  AGE     VERSION
k8s-calico-01   Ready    control-plane,master   36m     v1.20.0
k8s-calico-02   Ready    <none>                 10m     v1.20.0
k8s-calico-03   Ready    <none>                 3m10s   v1.20.0
[vagrant@k8s-calico-01 ~]$ kubectl get po,svc --all-namespaces
NAMESPACE     NAME                                           READY   STATUS             RESTARTS   AGE
kube-system   pod/calico-kube-controllers-7854b85cf7-nh5b9   1/1     Running            0          16m
kube-system   pod/calico-node-5hh4h                          0/1     Running            6          10m
kube-system   pod/calico-node-flkr5                          0/1     CrashLoopBackOff   7          16m
kube-system   pod/calico-node-m5qhf                          0/1     Running            1          3m26s
kube-system   pod/coredns-7f89b7bc75-fc4dd                   1/1     Running            0          36m
kube-system   pod/coredns-7f89b7bc75-sgkms                   1/1     Running            0          36m
kube-system   pod/etcd-k8s-calico-01                         1/1     Running            0          36m
kube-system   pod/kube-apiserver-k8s-calico-01               1/1     Running            0          36m
kube-system   pod/kube-controller-manager-k8s-calico-01      1/1     Running            0          22m
kube-system   pod/kube-proxy-7cwz9                           1/1     Running            0          10m
kube-system   pod/kube-proxy-mh97l                           1/1     Running            0          3m26s
kube-system   pod/kube-proxy-rxx9g                           1/1     Running            0          36m
kube-system   pod/kube-scheduler-k8s-calico-01               1/1     Running            0          22m

NAMESPACE     NAME                 TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)                  AGE
default       service/kubernetes   ClusterIP   10.96.0.1    <none>        443/TCP                  36m
kube-system   service/kube-dns     ClusterIP   10.96.0.10   <none>        53/UDP,53/TCP,9153/TCP   36m
[vagrant@k8s-calico-01 ~]$ kubectl get cs
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS    MESSAGE             ERROR
scheduler            Healthy   ok
controller-manager   Healthy   ok
etcd-0               Healthy   {"health":"true"}
[vagrant@k8s-calico-01 ~]$ kubectl get componentstatuses
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS    MESSAGE             ERROR
controller-manager   Healthy   ok
scheduler            Healthy   ok
etcd-0               Healthy   {"health":"true"}
[vagrant@k8s-calico-01 ~]$
```
#### 3.2.8 kube-proxy开启ipvs

修改kube-proxy的configmap，在config.conf中找到mode参数，改为mode: "ipvs"然后保存

注意: kube-proxy 开启ipvs 要求kernel version >= 4.1 , 否则会收到如下提示, 

```diff
- [vagrant@k8s-calico-01 ~]$ kubectl logs -f -l k8s-app=kube-proxy -n kube-system
- E0424 10:55:50.225402       1 proxier.go:389] can't set sysctl net/ipv4/vs/conn_reuse_mode, kernel version must be at least 4.1
- W0424 10:55:50.225489       1 proxier.go:445] IPVS scheduler not specified, use rr by default
- I0424 10:55:50.225784       1 server.go:650] Version: v1.20.0
```

```
# 查看kube-proxy 配置文件
[vagrant@k8s-calico-01 ~]$ kubectl get cm kube-proxy -o yaml -n kube-system

# 查看kube-proxy mode 配置 
[vagrant@k8s-calico-01 ~]$ kubectl get cm kube-proxy -o yaml -n kube-system  |grep mode
    mode: ""

# kube-proxy开启ipvs
[vagrant@k8s-calico-01 ~]$ kubectl get cm kube-proxy -o yaml -n kube-system |sed 's/mode: ""/mode: "ipvs"/g'| kubectl replace -f -
configmap/kube-proxy replaced

# 查看kube-proxy mode 模式
[vagrant@k8s-calico-01 ~]$ kubectl get cm kube-proxy -o yaml -n kube-system  |grep mode
    mode: "ipvs"

# 重启kube-proxy 
[vagrant@k8s-calico-01 ~]$ kubectl get pods -l k8s-app=kube-proxy -n kube-system
NAME               READY   STATUS    RESTARTS   AGE
kube-proxy-7cwz9   1/1     Running   0          20m
kube-proxy-mh97l   1/1     Running   0          13m
kube-proxy-rxx9g   1/1     Running   0          45m
[vagrant@k8s-calico-01 ~]$ kubectl delete pods -l k8s-app=kube-proxy -n kube-system
pod "kube-proxy-7cwz9" deleted
pod "kube-proxy-mh97l" deleted
pod "kube-proxy-rxx9g" deleted

# 确认ipvs模式开启成功
[vagrant@k8s-calico-01 ~]$ kubectl logs -f -l k8s-app=kube-proxy -n kube-system |grep ipvs
I1026 04:11:46.474911       1 server_others.go:176] Using ipvs Proxier.
I1026 04:11:42.842141       1 server_others.go:176] Using ipvs Proxier.
I1026 04:11:46.198116       1 server_others.go:176] Using ipvs Proxier.
```

因为部署的centos7.9.2009 内核版本是3.10的，所以没有开启成功，再下来升级内核

```
# 确保内核开启了ipvs模块
[vagrant@k8s-calico-01 ~]$ lsmod |grep ip_vs
ip_vs_sh               12688  0
ip_vs_wrr              12697  0
ip_vs_rr               12600  4
ip_vs                 145497  10 ip_vs_rr,ip_vs_sh,ip_vs_wrr
nf_conntrack          139264  9 ip_vs,nf_nat,nf_nat_ipv4,nf_nat_ipv6,xt_conntrack,nf_nat_masquerade_ipv4,nf_conntrack_netlink,nf_conntrack_ipv4,nf_conntrack_ipv6
libcrc32c              12644  4 xfs,ip_vs,nf_nat,nf_conntrack

# 升级内核
# 查看内核版本
[vagrant@k8s-calico-01 ~]$ cat /proc/version
Linux version 3.10.0-1127.el7.x86_64 (mockbuild@kbuilder.bsys.centos.org) (gcc version 4.8.5 20150623 (Red Hat 4.8.5-39) (GCC) ) #1 SMP Tue Mar 31 23:36:51 UTC 2020
[vagrant@k8s-calico-01 ~]$ uname -r
3.10.0-1127.el7.x86_64
[vagrant@k8s-calico-01 ~]$ uname -a
Linux k8s-calico-01 3.10.0-1127.el7.x86_64 #1 SMP Tue Mar 31 23:36:51 UTC 2020 x86_64 x86_64 x86_64 GNU/Linux
[vagrant@k8s-calico-01 ~]$ su
Password:
# 升级内核需要先导入elrepo的key，然后安装elrepo的yum源
[root@k8s-calico-01 vagrant]# rpm -import https://www.elrepo.org/RPM-GPG-KEY-elrepo.org
[root@k8s-calico-01 vagrant]# rpm -Uvh http://www.elrepo.org/elrepo-release-7.0-2.el7.elrepo.noarch.rpm
Retrieving http://www.elrepo.org/elrepo-release-7.0-2.el7.elrepo.noarch.rpm
Retrieving http://elrepo.org/elrepo-release-7.0-4.el7.elrepo.noarch.rpm
Preparing...                          ################################# [100%]
Updating / installing...
   1:elrepo-release-7.0-4.el7.elrepo  ################################# [100%]

# 查看内核相关的包
[root@k8s-calico-01 vagrant]# yum --disablerepo="*" --enablerepo="elrepo-kernel" list available
Loaded plugins: fastestmirror
Loading mirror speeds from cached hostfile
 * elrepo-kernel: hkg.mirror.rackspace.com
elrepo-kernel                                                          | 3.0 kB  00:00:00
elrepo-kernel/primary_db                                               | 2.0 MB  00:00:01
Available Packages
elrepo-release.noarch                        7.0-5.el7.elrepo          elrepo-kernel
kernel-lt.x86_64                             5.4.114-1.el7.elrepo      elrepo-kernel
kernel-lt-devel.x86_64                       5.4.114-1.el7.elrepo      elrepo-kernel
kernel-lt-doc.noarch                         5.4.114-1.el7.elrepo      elrepo-kernel
kernel-lt-headers.x86_64                     5.4.114-1.el7.elrepo      elrepo-kernel
kernel-lt-tools.x86_64                       5.4.114-1.el7.elrepo      elrepo-kernel
kernel-lt-tools-libs.x86_64                  5.4.114-1.el7.elrepo      elrepo-kernel
kernel-lt-tools-libs-devel.x86_64            5.4.114-1.el7.elrepo      elrepo-kernel
kernel-ml.x86_64                             5.11.16-1.el7.elrepo      elrepo-kernel
kernel-ml-devel.x86_64                       5.11.16-1.el7.elrepo      elrepo-kernel
kernel-ml-doc.noarch                         5.11.16-1.el7.elrepo      elrepo-kernel
kernel-ml-headers.x86_64                     5.11.16-1.el7.elrepo      elrepo-kernel
kernel-ml-tools.x86_64                       5.11.16-1.el7.elrepo      elrepo-kernel
kernel-ml-tools-libs.x86_64                  5.11.16-1.el7.elrepo      elrepo-kernel
kernel-ml-tools-libs-devel.x86_64            5.11.16-1.el7.elrepo      elrepo-kernel
perf.x86_64                                  5.11.16-1.el7.elrepo      elrepo-kernel
python-perf.x86_64                           5.11.16-1.el7.elrepo      elrepo-kernel

# 可以看出，长期维护版本lt为5.4.114，最新主线稳定版ml为5.11.16，我们需要安装最新的主线稳定内核，使用如下命令：(以后这台机器升级内核直接运行这句就可升级为最新稳定版)

# 安装最新的内核及内核开发包
[root@k8s-calico-01 vagrant]# yum -y --enablerepo=elrepo-kernel install kernel-ml.x86_64 kernel-ml-devel.x86_64
Loaded plugins: fastestmirror
Loading mirror speeds from cached hostfile
 * base: mirrors.163.com
 * elrepo: hkg.mirror.rackspace.com
 * elrepo-kernel: hkg.mirror.rackspace.com
 * extras: mirrors.163.com
 * updates: mirrors.tuna.tsinghua.edu.cn
elrepo                                                                  | 3.0 kB  00:00:00
elrepo/primary_db                                                       | 349 kB  00:00:00
Resolving Dependencies
--> Running transaction check
---> Package kernel-ml.x86_64 0:5.11.16-1.el7.elrepo will be installed
---> Package kernel-ml-devel.x86_64 0:5.11.16-1.el7.elrepo will be installed
--> Finished Dependency Resolution
...                                                            
Installed:
  kernel-ml.x86_64 0:5.11.16-1.el7.elrepo            kernel-ml-devel.x86_64 0:5.11.16-1.el7.elrepo

Complete!

# 查看系统所有安装的内核版本
[root@k8s-calico-01 vagrant]# awk -F\' '$1=="menuentry " {print $2}' /etc/grub2.cfg
CentOS Linux (5.11.16-1.el7.elrepo.x86_64) 7 (Core)
CentOS Linux (3.10.0-1127.el7.x86_64) 7 (Core)

# 查看系统当前系统的默认内核版本
[root@k8s-calico-01 vagrant]# grub2-editenv list
saved_entry=CentOS Linux (3.10.0-1127.el7.x86_64) 7 (Core)

# 重新设置默认内核版本
[root@k8s-calico-01 vagrant]# grub2-set-default 'CentOS Linux (5.11.16-1.el7.elrepo.x86_64) 7 (Core)'

# 查看默认内核版本
[root@k8s-calico-01 vagrant]# grub2-editenv list
saved_entry=CentOS Linux (5.11.16-1.el7.elrepo.x86_64) 7 (Core)

# 重启系统
[root@k8s-calico-01 vagrant]# reboot
Connection to 127.0.0.1 closed by remote host.
Connection to 127.0.0.1 closed.

# 再次查看内核版本
[vagrant@k8s-calico-01 ~]$ uname -r
5.11.16-1.el7.elrepo.x86_64

注意: 三节点内核都升级, 再次重启kube-proxy, 通过日志看到ipvs 成功开启

[vagrant@k8s-calico-01 ~]$ kubectl delete pods -l k8s-app=kube-proxy -n kube-system
pod "kube-proxy-6vkv5" deleted
pod "kube-proxy-hltst" deleted
pod "kube-proxy-j96gc" deleted

# ipvs 成功开启
[vagrant@k8s-calico-01 ~]$ kubectl logs -f -l k8s-app=kube-proxy -n kube-system |grep ipvs
I0424 11:59:40.373194       1 server_others.go:258] Using ipvs Proxier.
I0424 11:59:38.175708       1 server_others.go:258] Using ipvs Proxier.
I0424 11:59:38.767405       1 server_others.go:258] Using ipvs Proxier.
```

#### 3.2.9 部署Dashboard 
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
      nodePort: 30002
  selector:
    k8s-app: kubernetes-dashboard
```
3.创建service account并绑定默认cluster-admin管理员集群角色
```
[vagrant@k8s-calico-01 vagrant]$ ls
calico.yaml  init.sh  kubernetes-dashboard.yaml  nginx-deployment.yml  Vagrantfile
[vagrant@k8s-calico-01 vagrant]$ kubectl create serviceaccount dashboard-admin -n kube-system
serviceaccount/dashboard-admin created
[vagrant@k8s-calico-01 vagrant]$ kubectl create clusterrolebinding dashboard-admin --clusterrole=cluster-admin --serviceaccount=kube-system:dashboard-admin
clusterrolebinding.rbac.authorization.k8s.io/dashboard-admin created
```
4.查询dashboard token
```
[vagrant@k8s-calico-01 vagrant]$ kubectl describe secrets -n kube-system $(kubectl -n kube-system get secret | awk '/dashboard-admin/{print $1}')
Name:         dashboard-admin-token-nbt5h
Namespace:    kube-system
Labels:       <none>
Annotations:  kubernetes.io/service-account.name: dashboard-admin
              kubernetes.io/service-account.uid: a9fb94f1-9c4f-488c-a2c2-00f2f2c4a602

Type:  kubernetes.io/service-account-token

Data
====
ca.crt:     1066 bytes
namespace:  11 bytes
token:      eyJhbGciOiJSUzI1NiIsImtpZCI6Ijk3Z281UnJ6ODVzajJqX3JBYmF2TkUyeklhNENwVWxvWmdDRlE2V3ZfWGcifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJrdWJlLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJkYXNoYm9hcmQtYWRtaW4tdG9rZW4tbmJ0NWgiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC5uYW1lIjoiZGFzaGJvYXJkLWFkbWluIiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZXJ2aWNlLWFjY291bnQudWlkIjoiYTlmYjk0ZjEtOWM0Zi00ODhjLWEyYzItMDBmMmYyYzRhNjAyIiwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50Omt1YmUtc3lzdGVtOmRhc2hib2FyZC1hZG1pbiJ9.Dui_2p2-C0E_wDjye4q-2GNpltomclTvAtdx3gf79naAiZFvuLfhFaWmeeJTOq3hKNPzYuobP6HLMOL5V-9NBa0ZS4laN4vM5LK70JJm-I_ah94YLNkYjltxj_72ysQXfXb-bt1-1UNeJP1trT-dA7pNK8K2TxgT-DlOjjX9_FzmrkmjHoGILojQn4IEtiRogjWcz6kOuv_fgP6HILqHXrO13RwLLZAziX5MAD1m8ECfnjMm4oMmxUwtgdgpL2wtljSz0T32yzVipPYda16hb6TxIvFtQaUlRAahjDmdTNkQR7XbGMqO-yk6zzWS_0Tl385gDootKikzTtZE5ImmIw
```

5.创建kubernetes-dashboard 
```
创建kubernetes dashboard 
[vagrant@k8s-calico-01 vagrant]$ kubectl apply -f kubernetes-dashboard.yaml
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
[vagrant@k8s-calico-01 vagrant]$

# 查看kubernetes dashboard
[vagrant@k8s-calico-01 vagrant]$ kubectl get pod,svc -n kubernetes-dashboard
NAME                                             READY   STATUS    RESTARTS   AGE
pod/dashboard-metrics-scraper-7445d59dfd-k8jdl   1/1     Running   0          61s
pod/kubernetes-dashboard-7d8466d688-pt7hw        1/1     Running   0          61s

NAME                                TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)         AGE
service/dashboard-metrics-scraper   ClusterIP   10.110.160.214   <none>        8000/TCP        61s
service/kubernetes-dashboard        NodePort    10.106.98.32     <none>        443:30002/TCP   61s
```
6.访问dashboard 
```
http://192.168.206.10:30002 
通过前面生成的token访问dashboard 
```

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
[vagrant@k8s-calico-01 vagrant]$ kubectl expose deployment nginx-deployment --port=80 --type=NodePort
service/nginx-deployment exposed
[vagrant@k8s-calico-01 vagrant]$ kubectl get po,svc
NAME                                   READY   STATUS    RESTARTS   AGE
pod/nginx-deployment-585449566-4sqv7   1/1     Running   0          4m26s
pod/nginx-deployment-585449566-lqcn9   1/1     Running   0          4m26s

NAME                       TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
service/kubernetes         ClusterIP   10.96.0.1       <none>        443/TCP        130m
service/nginx-deployment   NodePort    10.107.237.54   <none>        80:30385/TCP   3s
```
查看曝露的服务, 发现对外网曝露的端口是30348，此时应该可以通过localhost:30348 访问页面了， 

访问nginx 页面
```
# 外网访问第一个节点
[vagrant@k8s-calico-01 vagrant]$ curl -I 192.168.206.10:30385
HTTP/1.1 200 OK
Server: nginx/1.19.10
Date: Sat, 24 Apr 2021 12:20:53 GMT
Content-Type: text/html
Content-Length: 612
Last-Modified: Tue, 13 Apr 2021 15:13:59 GMT
Connection: keep-alive
ETag: "6075b537-264"
Accept-Ranges: bytes

# 外网访问第二个节点
[vagrant@k8s-calico-01 vagrant]$ curl -I 192.168.206.11:30385
HTTP/1.1 200 OK
Server: nginx/1.19.10
Date: Sat, 24 Apr 2021 12:20:57 GMT
Content-Type: text/html
Content-Length: 612
Last-Modified: Tue, 13 Apr 2021 15:13:59 GMT
Connection: keep-alive
ETag: "6075b537-264"
Accept-Ranges: bytes

```
