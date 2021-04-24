
- [0.背景说明](#0背景说明)
- [1.面临问题](#1面临问题)
- [2.解决方案](#2解决方案)
- [3.集群搭建过程](#3集群搭建过程)
  - [3.1 集群模板搭建](#31-集群模板搭建)
    - [3.1.1 初始化虚拟机配置](#311-初始化虚拟机配置)
    - [3.1.2 修改虚拟机配置](#312-修改虚拟机配置)
    - [3.1.3 启动虚拟机](#313-启动虚拟机)
    - [3.1.4 导出为box模板 为后面复用](#314-导出为box模板-为后面复用)
## 0.背景说明

- 更新macOS Big Sur 后没法安装 minikube, 也没法通过Helm3 安装kubernetes 集群; 
- 之前安装过的集群因为占用磁盘空间所以没有使用后便删除了;

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

# k8s config
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



