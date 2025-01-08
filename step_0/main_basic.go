package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// go run main_basic.go run <cmd> <args>
// this command is similar to docker run <cmd> <args>
func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("help")
	}
}

// run runs the command in a new namespace and isolates this namespace from the host
func run() {
	fmt.Printf("Running %v \n", os.Args[2:])

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	must(cmd.Run())
}

// child is the child process that runs the command in the new namespace
func child() {
	fmt.Printf("Running %v \n", os.Args[2:])

	cg()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if _, err := os.Stat("/home/pj/ubuntufs"); os.IsNotExist(err) {
		panic("Root filesystem directory does not exist! Please create it before running the program.")
	}
	
	must(syscall.Sethostname([]byte("container")))
	must(syscall.Chroot("/home/pj/ubuntufs")) 
	// in order for the chroot setup to work , we need to create the directory and then populate it with the rootfs
	// the below commands can be used to create this setup using debootstrap
	// sudo apt-get install debootstrap 
	// sudo debootstrap stable /home/pj/ubuntufs http://deb.debian.org/debian/
	// or you can use alpine linux : just run these commands 
	// wget https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/x86_64/alpine-minirootfs-latest-x86_64.tar.gz
	// sudo tar -xzf alpine-minirootfs-latest-x86_64.tar.gz -C /home/pj/ubuntufs

	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	must(syscall.Mount("thing", "mytemp", "tmpfs", 0, ""))

	must(cmd.Run())

	must(syscall.Unmount("proc", 0))
	must(syscall.Unmount("thing", 0))
}

// cg sets up the cgroup - the control group responsible for limiting the resources of the container
func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(filepath.Join(pids, "pj"), 0755)
	must(ioutil.WriteFile(filepath.Join(pids, "pj/pids.max"), []byte("20"), 0700))
	// Removes the new cgroup in place after the container exits
	must(ioutil.WriteFile(filepath.Join(pids, "pj/notify_on_release"), []byte("1"), 0700))
	must(ioutil.WriteFile(filepath.Join(pids, "pj/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}