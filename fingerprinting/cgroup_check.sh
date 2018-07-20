#!/bin/bash
# look for docker cgroup or Cloudfoundry garden cgroup.
cat /proc/1/cgroup | grep -E -i ":/docker/|:/garden/"