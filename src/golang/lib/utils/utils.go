package utils

import (
	"os/user"
	"strconv"

	"github.com/shreyasksrao/jobmanager/lib/core"
)

func GetUidGidFromUserName(log core.Logger, userName string) (uid, gid int, err error) {
	log.Infof("Searching the user - %s", userName)
	usr, err := user.Lookup(userName)
	if err != nil {
		log.Errorf("Error while searching the User - %s", userName)
		return
	}
	uid, err = strconv.Atoi(usr.Uid)
	if err != nil {
		log.Errorf("Error while converting the user ID to int - %s", usr.Uid)
		return
	}
	log.Infof("Successfully fetched the UID of the User %s. UID - %v", userName, uid)
	gid, err = strconv.Atoi(usr.Gid)
	if err != nil {
		log.Errorf("Error while converting the group ID to int - %s", usr.Gid)
		return
	}
	log.Infof("Successfully fetched the GID of the User %s. GID - %v", userName, gid)
	return
}
