import subprocess

# entry point
if __name__ == '__main__':
  p_sd = subprocess.Popen(["../ceph/build/bin/radosgw",
                            "-d",
                            "--no-mon-config",
                            "--rgw-data", ".",
                            "--run-dir", ".",
                            "--rgw-sfs-data-path", ".",
                            "--rgw-backend-store", "sfs",
                            "--debug-rgw", "5",
                            "--rgw_frontends", "beast port=7482",
                            "--rgw_thread_pool_size", "1",
                            "--rgw_relaxed_region_enforcement", "1",
                            "--send-probe-evt-main", "false",
                            "--send-probe-evt-frontend-up", "false"],
                            cwd="../wd_sd")
  p_sd.wait()
