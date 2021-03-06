package service

import (
	"encoding/json"
	"fmt"

	"github.com/unix2dos/Journal/model"
)

type User struct {
}

func NewUser() *User {
	return &User{}
}

func (u *User) GetUserById(userId int64) (user *model.User, exist bool, err error) {

	key := u.getUserRedisKey(userId)
	user = new(model.User)

	exist, err = RedisStore.EXISTS(key)
	if err != nil {
		Logs.Errorf("EXISTS userId=%d err=%v", userId, err)
		return
	}

	if exist {
		//从redis找
		var str string
		str, err = RedisStore.Get(key)
		if err != nil {
			Logs.Errorf("HGET userId=%d err=%v", userId, err)
			return
		}
		if err = json.Unmarshal([]byte(str), user); err != nil {
			Logs.Errorf("HGET userId=%d err=%v", userId, err)
			return
		}
		return user, true, nil

	} else {
		//从数据库找
		exist, err = MysqlEngine.Id(userId).Get(user)
		if err != nil {
			Logs.Errorf("MysqlEngine Get userId=%d err=%v", userId, err)
			return
		}

		if exist {
			//写入redis
			u.SetUserToReids(user)
			return user, true, nil
		}
	}

	return
}

func (u *User) SetUserToReids(user *model.User) (err error) {
	key := u.getUserRedisKey(user.Id)
	byte, _ := json.Marshal(user)
	return RedisStore.Set(key, string(byte))
}

func (u *User) SetUserToMysqlAndRedis(user *model.User) (err error) {
	session := MysqlEngine.NewSession()
	session.Begin()
	defer func() {
		if err == nil {
			session.Commit()
		} else {
			session.Rollback()
		}
		session.Close()
	}()

	if user.LikeJournals == nil {
		user.LikeJournals = []int64{}
	}
	if user.LikeComments == nil {
		user.LikeComments = []int64{}
	}
	_, err = session.Insert(user)
	if err != nil { //insert错误, update一下, 如果update也错误,彻底错误
		session = session.MustCols("like_journals").MustCols("like_comments")
		_, err = session.ID(user.Id).Update(user)
		if err != nil {
			return
		}
	}

	err = u.SetUserToReids(user)
	return
}

//--------------------------------------------------//
func (u *User) getUserRedisKey(userId int64) string {
	return fmt.Sprintf(model.RedisKeyUser, userId)
}
