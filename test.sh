sudo apt remove libcudart10.1 nvidia-cuda-dev nvidia-cuda-doc nvidia-cuda-gdb nvidia-cuda-toolkit
sudo apt autoremove
...
# run cmake and specify the full path to nvcc
cmake .. -DLLAMA_CUBLAS=1 -DCMAKE_CUDA_COMPILER=/usr/local/cuda-12.2/bin/nvcc
