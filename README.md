# VMware ESXi vm clone
## I've prepared it because was too lazy to start vCenter when wanted to copy vm 
... not sure if will work properly when vcenter is started and ESXi is part of Host/Cluster vCenter configuration <br><br>
All variables (as user/pass and server name) must be configured as environment variables and SSHd in ESXi to be enabled for connection <br><br>
To use it after compilling / go build vm_Clone.go /:<br> 
<code> vmClone <old_machine_name> <new_machine_name> </code> 
