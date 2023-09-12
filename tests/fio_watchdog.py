import subprocess
import time

# entry point
if __name__ == '__main__':
  while True:
    p = subprocess.Popen(["fio",
                          "workload_1.fio"],
                          cwd=".")
    time.sleep(10)
    p.kill()
    p.wait()
