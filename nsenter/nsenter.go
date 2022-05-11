package nsenter

/*
#cgo CFLAGS: -Wall
#define _GNU_SOURCE
// #include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <fcntl.h>
#include <string.h>
#include <unistd.h>

__attribute__((constructor)) void enter_namespace(void) {
  char *minidocker_pid;
  // 从环境变量中获取需要进入的pid
  minidocker_pid = getenv("minidocker_pid");
  if (minidocker_pid) {
    // fprintf(stdout, "get minidocker_pid=%s\n", minidocker_pid);
  } else {
    // fprintf(stdout, "missing minidocker_pid env skip nsenter");
    return;
  }
  char *minidocker_command;
  // 从环境变量获取需要执行的命令
  minidocker_command = getenv("minidocker_command");
  if (minidocker_command) {
    // fprintf(stdout, "get minidocker_command=%s\n", minidocker_pid);
  } else {
    // fprintf(stdout, "missing minidocker_command env skip nsenter");
    return;
  }
  int i;
  char nspath[1024];
  // 需要进入的五种namespace
  char *namespaces[] = {"ipc", "uts", "net", "pid", "mnt"};

  for (i = 0; i < 5; i ++) {
    // 拼接对应的路径 /proc/pid/ns/ipc
    sprintf(nspath, "/proc/%s/ns/%s", minidocker_pid, namespaces[i]);
    printf("nspath: %s\n", nspath);
    int fd = open(nspath, O_RDONLY);
    // 调用setns系统调用进入对应的namespace
    if (setns(fd, 0) == -1) {
      printf("failed to set the process %s to namespace %s\n", minidocker_pid, namespaces[i]);
      // fprintf(stderr, "setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
    } else {
      // fprintf(stdout, "setns on %s namespace succeded\n", namespaces[i]);
    }
    close(fd);
  }
  exit(system(minidocker_command));
  return;
}
*/
import "C"

