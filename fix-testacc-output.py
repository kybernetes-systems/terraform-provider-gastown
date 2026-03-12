import re
with open("demo.md", "r") as f:
    data = f.read()

data = re.sub(r'Running acceptance tests with signal trapping...\ntrap: SIGINT: bad trap\n', 'Running acceptance tests with signal trapping...\n', data)
data = re.sub(r'make: \*\*\* \[Makefile:8: testacc\] Terminated\n', '', data)

with open("demo.md", "w") as f:
    f.write(data)
