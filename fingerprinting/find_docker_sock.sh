#!/bin/bash
find "/" 2>&1 | grep -E "(.*\/docker\.sock|^docker\.sock)$"