package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

type SshCred struct {
	src     string
	dst     string
	sPath   string
	client  *ssh.Client
	session *ssh.Session
}

var (
	passWord   = os.Getenv("PASSWORD") // passwd added as environment variable : Example: password
	userName   = os.Getenv("USERNAME") // username added as environment variable usualy root 
	serverName = os.Getenv("URL")      // servername added as environment variable : Example: 192.168.253.100

	port      string = "22"
	dataStore string = "/vmfs/volumes/datastore1/"
)

func SshInteractive(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	answers = make([]string, len(questions))
	for n, _ := range questions {
		answers[n] = passWord
	}

	return answers, nil
}

func (s *SshCred) conSSHserv() {
	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.KeyboardInteractive(SshInteractive),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshHst := serverName + ":" + port
	var err error
	s.client, err = ssh.Dial("tcp", sshHst, config)
	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}
}

func (s *SshCred) newSession() {
	s.conSSHserv()
	var err error
	s.session, err = s.client.NewSession()
	if err != nil {
		log.Fatalln("Failed to create New session: " + err.Error())
	}
}

func (s *SshCred) withOutpSSH() {

	comm := []string{
		"vim-cmd vmsvc/snapshot.get $(vim-cmd vmsvc/getallvms | grep " + s.src + " | awk {'print $1'})\ncheckSnapshot",
		"grep -e displayName $(find " + dataStore + " -type f \\( -perm 755 -o -perm 700 \\) | grep -ve \"~$\")\nsPathSet",
	}
	for _, v := range comm {
		s.newSession()
		defer s.session.Close()
		strs := strings.SplitAfterN(string(v), "\n", 2)
		z := strings.Trim(strs[0], "\n")

		xout, err := s.session.Output(z)
		if err != nil {
			log.Fatalln("Failed to execute cmd fot Output... " + err.Error())
		}

		if strs[1] == "checkSnapshot" {
			if strings.Contains(string(xout), `Snapshot Name`) {
				log.Fatalln("There is/are snapshots for " + s.src + " , please remove snapshots before to copy machine")
			}
		}
		if strs[1] == "sPathSet" {
			slOut := strings.Split(string(xout), "\n")
			var sCont []string
			for _, pv := range slOut {
				_, a, _ := strings.Cut(pv, "\"")
				b, _, _ := strings.Cut(a, "\"")

				if strings.Compare(b, s.src) == 0 {
					sCont = strings.Split(pv, "\"")
				}
			}
			s.sPath = strings.TrimSuffix(filepath.Base(strings.Split(sCont[0], ":")[0]), filepath.Ext(filepath.Base(strings.Split(sCont[0], ":")[0]))) // file path or name used in ESXi
		}
	}
}

func (s *SshCred) runSSH() {

	comm := []string{
		"vim-cmd vmsvc/snapshot.get $(vim-cmd vmsvc/getallvms | grep " + s.src + " | awk {'print $1'}) ",
		"vim-cmd vmsvc/power.getstate $(vim-cmd vmsvc/getallvms | grep " + s.src + " | awk {'print $1'}) | grep 'Powered off' && cp -r " + dataStore + s.sPath + " " + dataStore + s.dst,
		"vmkfstools -E " + dataStore + "/" + s.dst + "/" + s.sPath + ".vmdk " + dataStore + "/" + s.dst + "/" + s.dst + ".vmdk",
		"mv " + dataStore + "/" + s.dst + "/" + s.sPath + ".vmx " + dataStore + "/" + s.dst + "/" + s.dst + ".vmx",
		"sed 's/" + s.sPath + "/" + s.dst + "/g;s/" + s.src + "/" + s.dst + "/g' -i " + dataStore + s.dst + "/" + s.dst + ".vmx ",
		"IFS=$'\n' ; for f in $( find " + dataStore + s.dst + " -type f -name \"*" + s.sPath + "*\") ; do mv $f  $(echo ${f} | sed 's/" + s.sPath + "/" + s.dst + "/g') ; done",
		"vim-cmd solo/registervm " + dataStore + s.dst + "/" + s.dst + ".vmx",
	}
	for _, v := range comm {
		s.newSession()
		defer s.session.Close()
		err := s.session.Run(v)
		if err != nil {
			log.Fatalf("Exception in execute: \"%s\"", v)
		}
	}
}

func main() {
	var s SshCred
	if len(passWord) == 0 {
		fmt.Println("password is not set as PASSWORD environment variable")
		os.Exit(1)
	}
	if len(userName) == 0 {
		fmt.Println("user name is not set as USERNAME environment variable")
		os.Exit(1)
	}
	if len(serverName) == 0 {
		fmt.Println("URL is not set as URL environment variable")
		os.Exit(1)
	}
	if len(os.Args) != 3 {
		fmt.Println("To clone VM, please use: \"<FILE> Source_Name_VM Destination_Name_VM\"")
		log.Fatal("not enought args !!!! ")
	}
	s.src = os.Args[1]
	s.dst = os.Args[2]

	s.withOutpSSH()
	s.runSSH()
	defer s.client.Close()
}
