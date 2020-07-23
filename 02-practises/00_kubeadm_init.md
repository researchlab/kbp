1.引导前的检查

kubeadm init执行后，首先需要对集群master节点安装的各种约束条件进行逐一检查。

如果不符合kubeadm的要求，kubeadm将报错并停止init过程。

下面列举一些error级别的检查：
```
kubeadm版本要与安装的kubernetes版本的比对检查。
kubernetes安装的系统需求检查。
其它检查：用户、主机、端口、swap、工具等。
```

2.生成私钥和数字证书

kubeadm会为整个集群生成多组私钥和公钥数字证书。

包括整个集群的root CA的私钥和CA的公钥数字证书，

API Server与其它组件之间的相互通信所用的多组私钥和数字证书，

用于service acount token签名的私钥和公钥文件等。

这样使得kubeadm搭建出来的集群是一个安全的集群。

所有kubernetes自身组件的通信以及pod到API Service的通信都是基于安全数据通道的。

且有生成验证和授权机制的。

关于数字证书、CA和CA证书：

数字证书：互联网通讯中标志通讯各方身份信息的一串数字，提供了一种在互联网上验证通信实体身份的方式。

CA：Certificate Authority，证书授权中心。它是负责管理和签发证书的第三方机构。

　　它是负责管理和签发证书的第三方机构，作用是检查证书持有者身份的合法性，并签发证书，以访证书被伪造或篡改。

　　数字证书就是CA发行。

CA证书：CA颁发的证书，也就是我们常说的数字证书，包含证书拥有者的身份信息，CA机构的签名，公钥和私钥。

　　　身份信息用于证明证书持有者的身份；CA签名用于保护身份的真实性。公钥和私钥用于通信过程中加解密，从而保证通信信息的安全性。

查看kubeadm的证书：

主要包括：

（1）自建CA、生成ca.key和ca.crt

如果不指定外部的证书授权机构，那么kubeadm会自建证书授权机构，

生成私钥（ca.key）和自签署的数字证书（ca.crt）,用于后续签发kubernetes集群所需要的其它公钥证书证书。

查看ca的数字证书：

ca.crt是一个标准的x509格式的数字证书文件。
```
root@k8s1:/etc/kubernetes/pki# openssl x509 -in ca.crt -noout -text
Certificate:
    Data:
        Version: 3 (0x2)       #版本号
        Serial Number: 0 (0x0) #序列号， ca的第一个证书
    Signature Algorithm: sha256WithRSAEncryption   #加密方式
        Issuer: CN=kubernetes
        Validity
            Not Before: Jul 19 07:08:03 2020 GMT
            Not After : Jul 17 07:08:03 2030 GMT
        Subject: CN=kubernetes
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                Public-Key: (2048 bit)
                Modulus:
                    00:c2:6e:4d:00:ef:2f:ce:52:38:dd:53:53:87:21:
                    25:dd:b5:05:44:9c:57:16:c5:4a:92:ef:6e:9a:08:
                    1a:e0:8d:ab:6a:3c:13:86:5b:be:b7:f0:fc:98:dd:
                    dc:ce:f5:bb:d0:ee:ed:a3:ce:a2:b3:3a:29:1f:c0:
                    fd:90:c0:39:81:88:6b:74:af:36:49:9c:30:b9:cb:
                    67:2b:f2:f6:68:0b:8a:66:f6:fa:ad:54:e5:b1:1d:
                    7c:e2:4e:1f:8d:02:79:75:0f:96:e9:17:6b:c7:e1:
                    7a:4d:0f:4a:0c:f4:eb:92:5f:2a:4b:48:8d:e6:dc:
                    60:f1:28:6e:e9:a2:0f:e0:50:89:b9:56:ac:1f:f1:
                    5e:6a:cc:10:2f:5e:47:38:35:f5:bb:a2:30:87:30:
                    65:47:9c:60:28:92:b3:6a:bc:97:c5:ab:4f:69:af:
                    78:2f:d8:5c:4c:7a:ed:33:06:14:62:ea:0e:dd:af:
                    6b:44:58:74:d9:04:bc:4e:37:09:95:72:c0:2e:58:
                    17:24:88:e7:af:a6:3c:53:bc:1a:7c:7c:11:2a:d6:
                    fc:e0:0f:c0:83:b5:56:04:5e:0e:a6:b3:f5:f5:4c:
                    78:22:4d:19:93:4d:60:15:cf:75:a3:3e:fd:f2:10:
                    10:08:dc:86:3e:f3:67:26:cf:fb:ef:eb:31:e0:8f:
                    31:35
                Exponent: 65537 (0x10001)
        X509v3 extensions:  #证书的用途
            X509v3 Key Usage: critical  
                Digital Signature (数字签名), Key Encipherment (密钥加密), Certificate Sign (证书签发)
            X509v3 Basic Constraints: critical
                CA:TRUE   #这是一个ca的公钥证书
    Signature Algorithm: sha256WithRSAEncryption
         98:6a:2b:3c:6c:ff:43:67:ad:2f:91:0c:b7:9e:7a:4d:82:a2:
         32:3b:55:4f:37:55:a2:9c:8c:33:bb:1a:91:d3:0d:06:3d:22:
         86:9e:e1:2e:ce:d6:d5:80:c1:59:da:e1:18:5e:d0:5e:01:39:
         8c:d7:25:ec:8b:56:ca:bb:35:de:48:4a:7b:90:20:53:a6:bc:
         94:7f:bf:70:65:01:57:a5:3c:c9:f7:8e:b6:c9:6d:9d:60:32:
         ac:f8:97:27:09:95:07:37:25:b1:00:b2:08:f6:66:79:ab:7e:
         d9:21:c9:bf:f7:06:63:7c:c6:f5:d0:47:66:a1:bf:24:93:d9:
         9f:65:db:33:d8:4e:36:87:56:a4:48:89:f4:bb:11:aa:bb:0a:
         c5:8e:41:1b:7a:f7:6b:c5:0c:cc:65:8e:43:f6:5b:19:c2:36:
         ba:e1:e6:a8:fc:41:c5:99:2e:4e:88:fd:43:55:9b:81:04:cc:
         33:dc:a9:90:9f:a9:cd:d5:5a:38:d4:21:a0:6e:20:5c:b1:87:
         ca:0a:03:95:03:b4:c6:42:dc:9f:7e:5d:44:c1:e4:a5:e7:45:
         4a:18:b0:e2:f4:02:3a:71:73:45:6a:96:af:05:38:ea:d9:9d:
         85:b8:cf:bb:33:e2:ba:da:93:e8:e0:22:1f:7f:10:19:50:ca:
         eb:a3:7e:ad
root@k8s1:/etc/kubernetes/pki#
```

（2）apiserver的私钥与公钥证书

有了ca之后，就可以用来签发集群所使用的各种证书。

kubeadm会生成API Server的私钥文件，运用ca来签发API Server的公钥数据证书。

在pki目录下，apiserver.key是APIServer的私钥文件。apiserver.crt是用ca来签署的APIServer的公钥数据证书。

查看apiserver.crt证书内容：
```
root@k8s1:/etc/kubernetes/pki# openssl x509 -in apiserver.crt -noout -text
Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number: 606578464868448918 (0x86affe666e2ba96)
    Signature Algorithm: sha256WithRSAEncryption
        Issuer: CN=kubernetes
        Validity
            Not Before: Jul 19 07:08:03 2020 GMT
            Not After : Jul 19 07:08:04 2021 GMT
        Subject: CN=kube-apiserver   #kube-apiserver
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                Public-Key: (2048 bit)
                Modulus:
                    00:a7:85:7a:ff:1d:cf:6e:7c:59:3c:a3:70:ca:07:
                    7c:6f:13:f3:16:a7:7f:45:d4:e1:e7:71:f7:3f:e3:
                    2c:29:31:b4:74:43:ef:43:8a:e7:97:e9:ac:c6:e0:
                    af:e8:2b:e0:6d:09:b8:94:3f:58:54:bc:92:11:43:
                    9a:63:5b:1a:fe:43:43:5d:c3:f1:72:64:ac:34:51:
                    44:c2:77:4a:20:d9:99:84:ee:02:30:6f:d4:46:77:
                    97:c5:47:bc:25:f3:3d:5c:37:94:f8:3e:a1:7c:67:
                    bb:76:4f:9d:b2:35:f6:23:5d:34:58:da:4a:e8:15:
                    3b:6b:74:1f:1d:bf:0d:0e:c2:43:64:31:08:24:1e:
                    e3:b2:9a:58:ed:03:a8:b5:62:c8:91:bf:a4:a4:3b:
                    25:a7:cc:41:3f:12:07:0f:da:c1:56:85:1c:c5:e2:
                    df:1a:e4:72:93:99:39:3a:5c:0c:5c:43:03:d2:4a:
                    05:10:83:1b:40:bd:4a:04:ab:1a:1d:23:85:83:c1:
                    50:6c:a4:3d:c1:a3:78:b4:0e:c9:2f:ed:b6:96:ce:
                    3d:63:7e:37:44:6b:ca:30:7d:6b:83:62:de:3c:b6:
                    c0:b6:c9:8c:94:77:5c:52:4c:2d:d5:20:b9:65:4c:
                    b5:f0:33:45:24:db:2f:28:97:cf:f9:3a:ab:fa:c1:
                    19:93
                Exponent: 65537 (0x10001)
        X509v3 extensions:
            X509v3 Key Usage: critical
                Digital Signature(数字签名), Key Encipherment (密钥加密） #并没有签署证书的用途
            X509v3 Extended Key Usage:
                TLS Web Server Authentication
            X509v3 Subject Alternative Name:  #可用于多种域名和IP地址;
                DNS:k8s1, DNS:kubernetes, DNS:kubernetes.default, DNS:kubernetes.default.svc, DNS:kubernetes.default.svc.cluster.local, IP Address:10.96.0.1, IP Address:10.0.2.4
    Signature Algorithm: sha256WithRSAEncryption
         44:d4:f5:36:55:35:7d:ce:81:d1:d9:26:84:e1:65:9b:93:95:
         45:9b:f1:0c:83:b5:39:1b:ed:0a:f1:a5:7d:c1:91:11:d7:9c:
         ce:bf:69:22:aa:2a:41:6b:17:86:89:fa:dd:49:e0:3a:51:db:
         eb:17:2b:2f:cc:8d:24:13:db:bd:ad:0c:3b:8d:cd:9d:76:72:
         83:fd:d8:bf:50:92:77:01:93:af:a8:1c:9f:31:ed:76:74:95:
         dd:e5:0f:fa:a5:7b:76:0b:c4:e6:9c:99:5a:29:9e:7e:e3:b8:
         54:f7:86:90:3e:f6:f6:e1:36:f1:75:5d:dd:6d:9f:c4:e7:f2:
         e7:18:16:68:31:98:f7:af:be:cd:03:c3:8c:97:5e:28:f1:2c:
         26:19:3f:25:0f:09:98:43:5b:1a:1d:5a:30:c0:b6:d6:7e:cb:
         41:90:54:dc:f9:a4:5a:7b:26:23:94:47:f4:78:a3:1a:99:82:
         f5:bc:fc:db:c0:b3:cb:86:0f:59:31:6a:c7:27:9f:ea:8f:39:
         57:a4:3f:e2:24:ee:de:c9:49:bf:0d:e6:07:2b:6c:66:f9:18:
         07:90:8d:29:b9:87:b0:49:76:41:ed:d7:e2:db:a0:b3:f1:08:
         70:b2:a9:f3:38:b5:d7:26:b9:61:cf:3e:dd:2a:64:08:83:eb:
         76:42:04:8c
root@k8s1:/etc/kubernetes/pki#
```

（3）apiserver访问kubelet使用的客户端私钥与证书

APIServer会向各个node的kubelet主动发起链接，并从kubelet获取日志。或attach到正在运行的pod中，或提供kubelet的端口转发功能。

kubelet会通过client端的SSL证书，来校验APIServer建立的链接。

apiserver-kubelet-client.crt和apiserver-kubelet-client.key用来证明APIServer的合法身份的。

（4）sa.key和sa.pub

sa是service acount的缩写。

sa.key用于对service acount token的数据签名。sa.pub是sa.key对应的公钥文件。

（5）Etcd相关私钥和数字证书

etcd是kubernetes整个集群的控制中心，而集群中唯一可以访问的组件是APIServer，

其它组件都是通过API Server的API存储数据的。

为建立etcd和API Server之间的安全数据通道，kubeadm init会生成APIServer访问etcd的相关私钥和证书。

apiserver-etcd-client.crt和apiserver-etcd-client.key是kubeadm init生成的APIServer用于访问etcd的私钥文件和公钥数字证书。用于etcd对于APIServer的身份验证所用。

在pki目录下面，还有一个etcd目录，专门用来保存与etcd相关的证书文件。

我们可以看到，etcd下面也有一个ca，那apiserver-etcd-client.crt这个数字证书是由谁来签发的了？测试一下
```
root@k8s1:/etc/kubernetes/pki# openssl verify -CAfile ca.crt ./apiserver-etcd-client.crt
./apiserver-etcd-client.crt: O = system:masters, CN = kube-apiserver-etcd-client
error 7 at 0 depth lookup:certificate signature failure  #报错了，证明并非是由pki目录下的ca签发
140562870437528:error:0407006A:rsa routines:RSA_padding_check_PKCS1_type_1:block type is not 01:rsa_pk1.c:103:
140562870437528:error:04067072:rsa routines:RSA_EAY_PUBLIC_DECRYPT:padding check failed:rsa_eay.c:705:
140562870437528:error:0D0C5006:asn1 encoding routines:ASN1_item_verify:EVP lib:a_verify.c:218:
root@k8s1:/etc/kubernetes/pki# openssl verify -CAfile etcd/ca.crt ./apiserver-etcd-client.crt
./apiserver-etcd-client.crt: OK
root@k8s1:/etc/kubernetes/pki#
```
3.生成控制平面组件kubeconfig文件，这些文件将用于组件之间通信鉴权使用

通过kubeadm init将master节点启动成功后，告知了kubeconfig的配置方法。

配置之后，kubectl就可以直接使用这些信息，访问kubernetes集群。

这些文件都是kubeconfig文件，由kubeadm init生成。

kubelet.conf被kubelet组件使用，用于访问APIServer。

scheduler.conf被scheduler组件所使用，用于访问APIServer。

controller-manager.conf被controller-manager组件所使用，用于访问APIServer。

admin.conf包含了整个集群的最高权限配置数据。

一旦配置了KUBECONFIG变量，kubectl就会使用KUBECONFIG变量所配置的信息。

Kubeconfig包含cluster、user和context信息。
```
root@k8s1:/etc/kubernetes/pki# kubectl config view
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: REDACTED   #ca信息
    server: https://10.0.2.4:6443          #服务地址
  name: kubernetes
contexts:  # 用来绑定user和cluster
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes   #定义用哪个user访问哪个cluster
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
    client-certificate-data: REDACTED
    client-key-data: REDACTED
root@k8s1:/etc/kubernetes/pki#
```
允许kubectl快速切换context，这样可以使用不同身份管理不同集群。
 
4.生成控制平面组件manifest文件

这些文件将会被master节点上的kubelet所读取，并启动master控制平面组件，以及维护这些控制平面组件的状态。

虽然kubeadm init将集群引导起来，但是有些问题你需要知道。

比如控制平面的组件是怎么启动起来的？如果重启了master节点，这些组件是否还会自动重启。

这些控制组件的参数如何调整，又如何生效了？

kubeadm init在启动过程中，会生成各个组件的manifests文件。

对应master上所有组件。控制平面组件以Static Pod形式运行。
```
root@k8s1:/etc/kubernetes/manifests# cat kube-scheduler.yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ""
  creationTimestamp: null
  labels:
    component: kube-scheduler
    tier: control-plane
  name: kube-scheduler
  namespace: kube-system
spec:
  containers:
  - command:
    - kube-scheduler
    - --leader-elect=true
    - --kubeconfig=/etc/kubernetes/scheduler.conf
    - --address=127.0.0.1
    image: k8s.gcr.io/kube-scheduler-amd64:v1.10.2
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: 127.0.0.1
        path: /healthz
        port: 10251
        scheme: HTTP
      initialDelaySeconds: 15
      timeoutSeconds: 15
    name: kube-scheduler
    resources:
      requests:
        cpu: 100m
    volumeMounts:
    - mountPath: /etc/kubernetes/scheduler.conf
      name: kubeconfig
      readOnly: true
  hostNetwork: true
  volumes:
  - hostPath:
      path: /etc/kubernetes/scheduler.conf
      type: FileOrCreate
    name: kubeconfig
status: {}
root@k8s1:/etc/kubernetes/manifests#
```

这些manifests文件都是控制平面组件的yaml文件。

master节点上的pod将读取这些文件，并启动控制平面组件。

与普通pod不同的是，这些pod是以静态pod的形式运行。

静态pod是由节点上的kubelet进程来管理的，不通过APIServer来管理，也不关联任何replication的控制器。

由kubelet进程自己来监控。当静态pod崩溃时，kubelet会重启这些pod。

静态pod始终绑定在一个kubelet上，并且始终运行在同一个节点上。

kubelet会自动为每一个静态pod在APIServer上创建一个镜像的pod。

我们可以通过APIserver查询这些pod，但是不能通过APIServer对这些镜像进行控制。

如果要刪除静态pod，可以直接将其对应的manifests下的yaml文件移除即可。

所有控制平面的静态pod都运行在kubesystem的名称空间下，并且使用主机网络。

kubelet读取manifests目录并管理各控制平台组件pod启停。

如果修改调整组件的yaml文件，kubelet也会监视到这些变化，并重启对应的pod，使其配置生效。
 
5.下载镜像，等待控制平面启动

默认情况下Kubeadm依赖kubelet下载镜像并启动static pod.默认从k8s.gcr.io上下载组件镜像。

kubeadm会一直探测并等待localhost:6443/healthz服务返回成功。

localhost:6443/healthz服务是APIServer的存活探针服务。

存活探针配置在manifests的配置文件中的。
```
vim kube-apiserver.yaml
    ...
    livenessProbe:
      failureThreshold: 8  #失败门槛次数
      httpGet:
        host: 10.0.2.4
        path: /healthz
        port: 6443
        scheme: HTTPS
      initialDelaySeconds: 15
      timeoutSeconds: 15  #超时时间
    ...
```

6.保存MasterConfig配置信息，也就是集群创建的初始信息。
 
7.设置Master标值

将当前节点设置为master节点，这样工作负载就不会调度到master节点上。
 
8.进行基于TLS的安全引导相关配置

为后续的worker节点的加入，进行基于TLS的安全引导相关配置。
 
9.安装DNS和kube-proxy插件

DNS插件安装后会处于pending状态，需要等到网络插件安装后才能恢复到running状态。

以DaemonSet方式部署Kube-proxy。
```
root@k8s1:~# kubectl get daemonset -n kube-system
NAME         DESIRED   CURRENT   READY     UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
kube-proxy   3         3         3         3            3           <none>          2d
weave-net    3         3         3         3            2           <none>          2d
root@k8s1:~#
```
部署Kube-dns插件，作为内部DNS服务，可以使用CoreDNS来替代。
 
