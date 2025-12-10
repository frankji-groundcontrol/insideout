# 安装anaconda
1. 进入https://www.anaconda.com/download/success
2. 下载对应版本
3. 安装环境
3.1 Windows打开之后会是一个安装向导，按照以下配置环境
3.2 macos
conda config --set custom_channels.auto https://mirrors.tuna.tsinghua.edu.cn/anaconda/cloud/
conda create -n webapp -c conda-forge -c anaconda nodejs yarn pnpm
4. 激活环境
conda activate webapp
