import subprocess

# entry point
if __name__ == '__main__':
  while True:
    p = subprocess.Popen(["../ceph/build/bin/radosgw",
                          "-d",
                          "--no-mon-config",
                          "--rgw-data", ".",
                          "--run-dir", ".",
                          "--rgw-sfs-data-path", ".",
                          "--rgw-backend-store", "sfs",
                          "--debug-rgw", "5",
                          "--rgw_thread_pool_size", "512",
                          "--probe-endpoint", "http://localhost:8080"],
                          cwd="../wd")
    p.wait()
