docker run -it -v /:/mnt --net host --ipc host --pid host --privileged ubuntu bash -c "chroot /mnt bash"
