package resolver

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
)

const (
	parseFormat               = "2006-01-02T15:04:05.000"
	defaultResolveIntervalSec = 30   // 重新resolve 一个表项的间隔时间30秒
	defaultResolveTimeoutSec  = 29   // 超过29秒未在链上看到这个表项认为已经resolved完成
	reservedOdds              = 0.98 // 预留2%的利润给自己
)

//var defaultRPCServer = "https://nodes.get-scatter.com:443"
type respDataDetail interface{}

// respGetActions :
type respGetActions struct {
	Actions []struct {
		AccountActionSeq int64 `json:"account_action_seq"`
		ActionTrace      struct {
			Act struct {
				Account       string `json:"account"`
				Authorization []struct {
					Actor      string `json:"actor"`
					Permission string `json:"permission"`
				} `json:"authorization"`
				Data    respDataDetail
				HexData string `json:"hex_data"`
				Name    string `json:"name"`
			} `json:"act"`
			Console      string        `json:"console"`
			CPUUsage     int           `json:"cpu_usage"`
			Elapsed      int           `json:"elapsed"`
			InlineTraces []interface{} `json:"inline_traces"`
			Receipt      struct {
				AbiSequence    int             `json:"abi_sequence"`
				ActDigest      string          `json:"act_digest"`
				AuthSequence   [][]interface{} `json:"auth_sequence"`
				CodeSequence   int             `json:"code_sequence"`
				GlobalSequence int             `json:"global_sequence"`
				Receiver       string          `json:"receiver"`
				RecvSequence   int             `json:"recv_sequence"`
			} `json:"receipt"`
			TotalCPUUsage int    `json:"total_cpu_usage"`
			TrxID         string `json:"trx_id"`
		} `json:"action_trace"`
		BlockNum        uint64 `json:"block_num"`
		BlockTime       string `json:"block_time"`
		GlobalActionSeq int    `json:"global_action_seq"`
	} `json:"actions"`
	LastIrreversibleBlock int `json:"last_irreversible_block"`
}

type respGetInfo struct {
	ServerVersion            string `json:"server_version"`
	HeadBlockNum             int64  `json:"head_block_num"`
	LastIrreversibleBlockNum int64  `json:"last_irreversible_block_num"`
	HeadBlockID              string `json:"head_block_id"`
	HeadBlockTime            string `json:"head_block_time"`
	HeadBlockProducer        string `json:"head_block_producer"`
}

type respError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   struct {
		Code    int           `json:"code"`
		Name    string        `json:"name"`
		What    string        `json:"what"`
		Details []interface{} `json:"details"`
	} `json:"error"`
}

// OneshotInfo :
type OneshotInfo struct {
	OsID     uint64 `json:"osid"`
	Player   string `json:"player"`
	Amt      int64  `json:"amt"`
	Celling  int64  `json:"celling"`
	MicroSec int64  `json:"microsec,string"`
}

// OneshotTbl :
type OneshotTbl struct {
	OneshotInfo
	Solved         bool
	LastSolveSec   int64
	LastOnchainSec int64
	BlkNum         uint64
	BlkHash        string
	DiceVal        uint8
}

type respOneshot struct {
	Rows []OneshotInfo `json:"rows"`
	More bool          `json:"more"`
}

// GroupItem :
type GroupItem struct {
	RltID    int64  `json:"rltid"`
	MicroSec int64  `json:"microsec,string"`
	Player   string `json:"player"`
}

// GroupTbl :
type GroupTbl struct {
	GroupItem
	Solved         bool
	LastSolveSec   int64
	LastOnchainSec int64
	GrpType        string
	BlkNum         uint64
	BlkHash        string
	DiceVal        uint64
}

type respGroup struct {
	More bool        `json:"more"`
	Rows []GroupItem `json:"rows"`
}

type respBlkInfo struct {
	respError
	Timestamp         string        `json:"timestamp"`
	Producer          string        `json:"producer"`
	Confirmed         int           `json:"confirmed"`
	Previous          string        `json:"previous"`
	TransactionMroot  string        `json:"transaction_mroot"`
	ActionMroot       string        `json:"action_mroot"`
	ScheduleVersion   int           `json:"schedule_version"`
	NewProducers      interface{}   `json:"new_producers"`
	HeaderExtensions  []interface{} `json:"header_extensions"`
	ProducerSignature string        `json:"producer_signature"`
	Transactions      []interface{} `json:"transactions"`
	BlockExtensions   []interface{} `json:"block_extensions"`
	ID                string        `json:"id"`
	BlockNum          uint64        `json:"block_num"`
	RefBlockPrefix    int64         `json:"ref_block_prefix"`
}

// BlockInfoTbl :
type BlockInfoTbl struct {
	BlockNum  uint64 // block number
	MicroSec  int64  // block produced @ when
	TmStr     string // block produced @ when
	BlockHash string // block id/hash
	From      string // from which http access point
}

// DiceHistoryTbl :
type DiceHistoryTbl struct {
	ResolveDate   time.Time // 什么时间结算的
	OsID          uint64    // oneshot ID
	BlkNum        uint64    // 用以结算的blknum
	BlkID         string    // 用以结算的blk hash
	DiceVal       uint8     // 骰子值
	MicroSec      uint64    // resolve所用的block的microsec
	Celling       uint8     // 上限
	Player        string
	Result        string
	Bet           string
	Reward        string
	BetDate       string // 押注时间
	AccountActSeq uint64
}

// GroupHistoryTbl :
type GroupHistoryTbl struct {
	ResolveDate   time.Time // 什么时间结算的
	BlkNum        uint64
	BlkID         string
	DiceVal       uint64
	MicroSec      uint64
	Reward        string
	GrpType       string
	Winner        string
	WinnerID      uint64
	AccountActSeq uint64
}

// ResolverCtrl :
type ResolverCtrl struct {
	lock             sync.Mutex
	DbPath           string
	dbmap            *gorp.DbMap
	RpcURL           string
	WalletKey        string
	ContractServant  string
	FetchIdleDur     time.Duration // 查询blk时间间隔
	BPInterval       time.Duration // 出块间隔
	FromBlkNum       int64         // 从哪个blocknum开始查询
	hisHeadBlknum    int64         // 路标头节点头编号
	hisHeadBlkTime   time.Time     // 路标头节点的时间戳
	hisHeadBlkUptime time.Time     // 什么时候记录的路标头节点
	isForEosforce    bool
}

func (c *ResolverCtrl) initDB() error {
	if c.dbmap == nil {
		db, err := sql.Open("sqlite3", c.DbPath)
		if nil != err {
			log.Printf("initDB - open %s failed : %v", c.DbPath, err)
			return err
		}
		c.dbmap = &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
		if _, err = c.dbmap.Exec("PRAGMA synchronous=NORMAL"); nil != err {
			log.Printf("initDB - 'PRAGMA synchronous=NORMAL' failed : %v", err)
		}
		if _, err = c.dbmap.Exec("PRAGMA page_size=8192"); nil != err {
			log.Printf("initDB - 'PRAGMA page_size=8192' failed : %v", err)
		}
		if _, err = c.dbmap.Exec("PRAGMA cache_size=204800"); nil != err {
			log.Printf("initDB - 'PRAGMA cache_size=204800' failed : %v", err)
		}
		if _, err = c.dbmap.Exec("PRAGMA temp_store=MEMORY"); nil != err {
			log.Printf("initDB - 'PRAGMA temp_store=MEMORY' failed : %v", err)
		}

		c.dbmap.AddTableWithName(BlockInfoTbl{}, "BlockInfoTbl").SetKeys(false, "BlockNum")
		c.dbmap.AddTableWithName(OneshotTbl{}, "OneshotTbl").SetKeys(false, "OsID")
		c.dbmap.AddTableWithName(DiceHistoryTbl{}, "DiceHistoryTbl").SetKeys(false, "OsID")
		c.dbmap.AddTableWithName(GroupTbl{}, "GroupTbl").SetKeys(false, "RltID", "GrpType")
		c.dbmap.AddTableWithName(GroupHistoryTbl{}, "GroupHistoryTbl").SetKeys(false, "AccountActSeq")

		if err = c.dbmap.CreateTablesIfNotExists(); nil != err {
			log.Printf("initDB - CreateTablesIfNotExists failed : %v", err)
		}
	}
	return nil
}

// /usr/local/bin/cleos  --wallet-url http://127.0.0.1:8900  -u http://172.17.0.2:8888 get table diceonlineos grpa roulette
func (c *ResolverCtrl) fetchGroup(group string) (err error) {
	URL := c.RpcURL + "/v1/chain/get_table_rows"
	scope := ""
	if group == "group10" {
		scope = "grpa"
	} else if group == "group100" {
		scope = "grpb"
	} else {
		return fmt.Errorf("'%s' wrong group param", group)
	}

	paramsMap := make(map[string]interface{})
	paramsMap["code"] = c.ContractServant
	paramsMap["scope"] = scope
	paramsMap["table"] = "roulette"
	paramsMap["json"] = true
	paramsMap["limit"] = 100

	buf, err := json.Marshal(paramsMap)
	if nil != err {
		log.Printf("fetchGroup - json.Marshal failed : %v", err)
		return err
	}

	resp, err := http.Post(URL, "application/json", strings.NewReader(string(buf)))
	if nil != err {
		log.Printf("fetchGroup - http.Post failed : %v", err)
		return err
	}

	buf, err = ioutil.ReadAll(resp.Body)
	if nil != err {
		log.Printf("fetchGroup - ioutil.ReadAll failed : %v", err)
		return err
	}

	var rgp respGroup
	if err = json.Unmarshal(buf, &rgp); nil != err {
		log.Printf("fetchGroup - json.Unmarshall failed : %v", err)
		log.Printf("%s", string(buf))
		return err
	}

	if len(rgp.Rows) == 0 {
		//log.Printf("fetchGroup - no more rows now, sleep a while")
		return nil
	}

	log.Printf("fetchGroup - %s got [%d -> %d]", group, rgp.Rows[0].RltID, rgp.Rows[len(rgp.Rows)-1].RltID)
	now := time.Now()
	for idx := range rgp.Rows {
		grp := &rgp.Rows[idx]
		var old GroupTbl
		c.lock.Lock()
		err = c.dbmap.SelectOne(&old, "SELECT * FROM GroupTbl WHERE RltID=? AND GrpType=?", grp.RltID, group)
		if nil != err {
			if strings.Contains(err.Error(), "sql: no rows in result set") { // 区别对待no such row
				err = c.dbmap.Insert(&GroupTbl{
					GroupItem: GroupItem{
						RltID:    grp.RltID,
						MicroSec: grp.MicroSec,
						Player:   grp.Player,
					},
					Solved:         false,
					LastSolveSec:   0,
					LastOnchainSec: now.Unix(),
					GrpType:        group,
				})
				if nil != err {
					log.Printf("fetchGroup - c.dbmap.Exec insert failed : %v", err)
				}
			} else {
				log.Printf("fetchGroup - c.dbmap.SelectOne failed : %v", err)
				c.lock.Unlock()
				return err
			}
		} else { // 已经有这个数据了，更新 LastOnchainSec 和 Solved为false
			if _, err = c.dbmap.Exec("UPDATE GroupTbl SET LastOnchainSec=?,Solved=0 WHERE RltID=? AND GrpType=?", now.Unix(), grp.RltID, group); nil != err {
				log.Printf("fetchGroup - c.dbmap.Exec update %d failed : %v", grp.RltID, err)
			}
		}
		c.lock.Unlock()
		if nil != err {
			log.Printf("fetchOneshots - abort early since %v", err)
			return err
		}
	}
	return nil
}

// /usr/local/bin/cleos  --wallet-url http://127.0.0.1:8900  -u http://172.17.0.2:8888 get table diceonlineos oneshot oneshot
// curl --request POST --url http://127.0.0.1:8888/v1/chain/get_table_rows
func (c *ResolverCtrl) fetchOneshots() (err error) {
	URL := c.RpcURL + "/v1/chain/get_table_rows"

	paramsMap := make(map[string]interface{})
	paramsMap["code"] = c.ContractServant
	paramsMap["scope"] = "oneshot"
	paramsMap["table"] = "oneshot"
	paramsMap["json"] = true
	paramsMap["limit"] = 100

	buf, err := json.Marshal(paramsMap)
	if nil != err {
		log.Printf("fetchOneshots - json.Marshal failed : %v", err)
		return err
	}

	resp, err := http.Post(URL, "application/json", strings.NewReader(string(buf)))
	if nil != err {
		log.Printf("fetchOneshots - http.Post failed : %v", err)
		return err
	}

	buf, err = ioutil.ReadAll(resp.Body)
	if nil != err {
		log.Printf("fetchOneshots - ioutil.ReadAll failed : %v", err)
		return err
	}

	var ros respOneshot
	if err = json.Unmarshal(buf, &ros); nil != err {
		log.Printf("fetchOneshots - json.Unmarshall failed : %v", err)
		log.Printf("%s", string(buf))
		return err
	}

	if len(ros.Rows) == 0 {
		//log.Printf("fetchOneshots - no more rows now, sleep a while")
		time.Sleep(c.FetchIdleDur)
		return nil
	}

	//log.Printf("fetchOneshots - got [%d -> %d]", ros.Rows[0].OsID, ros.Rows[len(ros.Rows)-1].OsID)

	now := time.Now()

	for idx := range ros.Rows {
		row := &ros.Rows[idx]
		c.lock.Lock()
		var old OneshotTbl
		err = c.dbmap.SelectOne(&old, "SELECT * FROM OneshotTbl WHERE OsID = ?", row.OsID)
		if nil != err {
			if strings.Contains(err.Error(), "sql: no rows in result set") { // 区别对待no such row
				err = c.dbmap.Insert(&OneshotTbl{
					OneshotInfo: OneshotInfo{
						OsID:     row.OsID,
						Player:   row.Player,
						Amt:      row.Amt,
						Celling:  row.Celling,
						MicroSec: row.MicroSec,
					},
					Solved:         false,
					LastSolveSec:   0,
					LastOnchainSec: now.Unix(),
				})
				if nil != err {
					log.Printf("fetchOneshots - c.dbmap.Exec insert failed : %v", err)
				}
			} else {
				log.Printf("fetchOneshots - c.dbmap.SelectOne failed : %v", err)
				c.lock.Unlock()
				return err
			}
		} else { // 已经有这个数据了，更新 LastOnchainSec 和 Solved为false
			if _, err = c.dbmap.Exec("UPDATE OneshotTbl SET Player=?,Amt=?,Celling=?,MicroSec=?,LastOnchainSec=?,Solved=0 WHERE OsID=?", row.Player, row.Amt, row.Celling, row.MicroSec, now.Unix(), row.OsID); nil != err {
				log.Printf("fetchOneshots - c.dbmap.Exec update %d failed : %v", row.OsID, err)
			}
		}

		c.lock.Unlock()
		if nil != err {
			log.Printf("fetchOneshots - abort early since %v", err)
			return err
		}
	}
	return nil
}

func (c *ResolverCtrl) execCmd(args []string) (err error) {
	//cleos wallet unlock --password xxxxx
	unlockCmd := exec.Command("/usr/local/bin/cleos", "wallet", "unlock", "--password", c.WalletKey)
	stdoutStderr, err := unlockCmd.CombinedOutput()
	if err != nil && !strings.Contains(string(stdoutStderr), "Already unlocked") {
		log.Printf("ResolverCtrl.execCmd - unlockCmd failed, err : %v, output:%s\n", err, string(stdoutStderr))
		return err
	}
	//log.Printf("ResolverCtrl.execCmd - unlockCmd done : %s\n", string(stdoutStderr))

	defer func() { //cleos wallet lock
		lockCmd := exec.Command("/usr/local/bin/cleos", "wallet", "lock")
		stdoutStderr, err := lockCmd.CombinedOutput()
		if err != nil {
			log.Printf("ResolverCtrl.execCmd - lockCmd failed, err : %v, output:%s\n", err, string(stdoutStderr))
		}
	}()

	cmd := exec.Command("/usr/local/bin/cleos", args...)
	stdoutStderr, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("ResolverCtrl.execCmd - cmd failed, err : %v, output:%s\n", err, string(stdoutStderr))
		return err
	}
	//log.Printf("ResolverCtrl.execCmd - cmd done : %s\n", string(stdoutStderr))
	return nil
}

func (c *ResolverCtrl) doSolveGroup(group string, rangeBeginID int64) (err error) {
	var reward string
	var base int64

	if group == "group10" {
		reward = "9.5000 EOS"
		base = 10
	} else if group == "group100" {
		reward = "95.0000 EOS"
		base = 100
	} else {
		return fmt.Errorf("ResolverCtrl.doSolveGroup - invalid group : %s", group)
	}

	var blk BlockInfoTbl
	var rangeLast GroupTbl
	c.lock.Lock()
	err = c.dbmap.SelectOne(&rangeLast, "SELECT * FROM GroupTbl WHERE RltID=? AND GrpType=?", rangeBeginID+base-1, group)
	if nil != err {
		c.lock.Unlock()
		log.Printf("ResolverCtrl.doSolveGroup - failed to select range last %d, err : %v", rangeBeginID+base-1, err)
		return err
	}

	err = c.dbmap.SelectOne(&blk, "SELECT * FROM BlockInfoTbl WHERE MicroSec > ? ORDER BY BlockNum ASC LIMIT 1", rangeLast.MicroSec)
	c.lock.Unlock()
	if nil != err {
		if !strings.Contains(err.Error(), "sql: no rows in result set") {
			log.Printf("ResolverCtrl.doSolveGroup - c.dbmap.SelectOne failed : %v", err)
			log.Printf("SELECT * FROM BlockInfoTbl WHERE MicroSec > %d ORDER BY BlockNum ASC LIMIT 1", rangeLast.MicroSec)
		}

		time.Sleep(c.FetchIdleDur)
		return err
	}

	log.Printf("ResolverCtrl.doSolveGroup - use blk %d @ %s %d to solve group %d @ %d", blk.BlockNum, blk.TmStr, blk.MicroSec, rangeLast.RltID, rangeLast.MicroSec)

	buf := fmt.Sprintf("%s", blk.BlockHash)
	hash := sha256.Sum256([]byte(buf))
	diceval := uint64(0)
	for _, val := range hash {
		diceval += uint64(val)
	}

	winnerID = int64(diceval) + rangeBeginID
	var winner GroupTbl
	c.lock.Lock()
	err = c.dbmap.SelectOne(&winner, "SELECT * FROM GroupTbl WHERE RltID=? AND GrpType=?", winnerID, group)
	if nil != err {
		c.lock.Unlock()
		log.Printf("ResolverCtrl.doSolveGroup - failed to select winner @ %d, err : %v", winnerID, err)
		return err
	}
	c.lock.Unlock()

	log.Printf("ResolverCtrl.doSolveGroup - %s => %x => %d => %d", blk.BlockHash, hash, diceval, diceval%100)
	diceval %= uint64(base)
	commentMap := make(map[string]interface{})
	commentMap["reward"] = reward
	commentMap["winner"] = winner.Player
	commentMap["winnerID"] = winnerID

	tmpBuf, _ := json.Marshal(commentMap)
	comment := strings.Replace(string(tmpBuf), `"`, `\"`, -1)

	args := []string{"-u", c.RpcURL,
		"push", "action", c.ContractServant, "resolvegrp",
		fmt.Sprintf(`{"blknum":%d,"microsec":%d,"blkid":"%s","diceval":%d,"forgrp":%d,"grpbase":%d,"comment":"%s"}`, blk.BlockNum, blk.MicroSec, blk.BlockHash, diceval, base, rangeLast.RltID-base+1, comment),
		"-p", c.ContractServant}

	if err = c.execCmd(args); nil == err {
		c.lock.Lock()
		_, err = c.dbmap.Exec("UPDATE GroupTbl SET LastSolveSec=?,BlkNum=?,BlkHash=?,DiceVal=? WHERE RltID>=? AND RltID<=? AND GrpType=?",
			time.Now().Unix(),
			blk.BlockNum,
			blk.BlockHash,
			diceval,
			rangeLast.RltID-base+1,
			rangeLast.RltID,
			group)
		c.lock.Unlock()
	}
	if nil != err {
		log.Printf("ResolverCtrl.doSolveGroup - execCmd or Exec failed : %v", err)
	}
	return err
}

func (c *ResolverCtrl) doSolveOneshot(os *OneshotInfo) (err error) {
	var blk BlockInfoTbl

	c.lock.Lock()
	err = c.dbmap.SelectOne(&blk, "SELECT * FROM BlockInfoTbl WHERE MicroSec > ? ORDER BY BlockNum ASC LIMIT 1", os.MicroSec)
	c.lock.Unlock()
	if nil != err {
		if !strings.Contains(err.Error(), "sql: no rows in result set") {
			log.Printf("c.doSolveOneshot - c.dbmap.SelectOne failed : %v", err)
			log.Printf("SELECT * FROM BlockInfoTbl WHERE strftime('%%s', TmStr) > %d ORDER BY BlockNum ASC LIMIT 1", os.MicroSec)
		}
		time.Sleep(time.Millisecond * 100)
		return err
	}

	log.Printf("c.doSolveOneshot - use blk %d @ %s %d to solve oneshot %d @ %d", blk.BlockNum, blk.TmStr, blk.MicroSec, os.OsID, os.MicroSec)

	buf := fmt.Sprintf("%s%d", blk.BlockHash, os.OsID)
	hash := sha256.Sum256([]byte(buf))
	diceval := uint64(0)
	for _, val := range hash {
		diceval += uint64(val)
	}

	log.Printf("c.doSolveOneshot - %s%d => %x => %d => %d", blk.BlockHash, os.OsID, hash, diceval, 1+(diceval%100))
	diceval = 1 + (diceval % 100) // [0,99] ==> [1,100]

	commentMap := make(map[string]interface{})

	commentMap["player"] = os.Player
	commentMap["celling"] = os.Celling
	commentMap["bet"] = fmt.Sprintf("%.4f EOS", float64(os.Amt)/10000.0)
	commentMap["betDate"] = time.Unix(os.MicroSec/1000000, 1000*(os.MicroSec%1000000)).Format(time.RFC3339)

	if int64(diceval) < os.Celling {
		commentMap["reward"] = fmt.Sprintf("%.4f EOS", float64(os.Amt)*calcodds(os.Celling)/10000.0)
	} else {
		commentMap["reward"] = "0.0000 EOS"
	}

	if int(diceval) < int(os.Celling) {
		commentMap["res"] = "win"
		commentMap["reward"] = fmt.Sprintf("%.4f EOS", float64(os.Amt)*calcodds(os.Celling)/10000.0)
	} else {
		commentMap["res"] = "lost"
		commentMap["reward"] = "0.0001 EOS"
	}
	tmpBuf, _ := json.Marshal(commentMap)
	comment := strings.Replace(string(tmpBuf), `"`, `\"`, -1)
	args := []string{"-u", c.RpcURL,
		"push", "action", c.ContractServant, "resolveos",
		fmt.Sprintf(`{"osid":%d,"blknum":%d,"microsec":%d,"diceval":%d,"blkid":"%s","comment":"%s"}`, os.OsID, blk.BlockNum, blk.MicroSec, diceval, blk.BlockHash, comment),
		"-p", c.ContractServant}

	if err = c.execCmd(args); nil != err {
		log.Printf("c.doSolveOneshot - execCmd or Exec failed : %v\n\tcommand cleos %s", err, strings.Join(args, " "))
	}
	c.lock.Lock()
	_, err = c.dbmap.Exec("UPDATE OneshotTbl SET LastSolveSec=?,BlkNum=?,BlkHash=?,DiceVal=? WHERE OsID=?",
		time.Now().Unix(),
		blk.BlockNum,
		blk.BlockHash,
		diceval,
		os.OsID)
	c.lock.Unlock()

	return err
}

func (c *ResolverCtrl) solveOneshotsRoutine() {
	for {
		var oss []OneshotTbl
		now := time.Now()

		c.lock.Lock()

		// LastOnchainSec 超过29秒以上的oneshot项，认为已经结算完成
		_, err := c.dbmap.Exec("UPDATE OneshotTbl SET Solved=1 WHERE Solved=0 AND LastOnchainSec<?", now.Unix()-defaultResolveIntervalSec)
		if nil != err {
			log.Printf("c.solveOneshotsRoutine - c.dbmap.Exec failed : %v", err)
		}

		// 只结算30秒以内未处理过的oneshot项目，
		_, err = c.dbmap.Select(&oss, "SELECT * FROM OneshotTbl WHERE Solved=0 AND LastSolveSec<? ORDER BY OsID ASC", now.Unix()-defaultResolveIntervalSec)
		c.lock.Unlock()

		if nil != err {
			log.Printf("ResolverCtrl.solveOneshotsRoutine - c.dbmap.Select failed : %v", err)
			time.Sleep(c.FetchIdleDur)
			continue
		}
		if len(oss) == 0 {
			time.Sleep(c.FetchIdleDur)
			continue
		}

		log.Printf("ResolverCtrl.solveOneshotsRoutine - %d to be solved, [%d-%d]", len(oss), oss[0].OsID, oss[len(oss)-1].OsID)

		for idx := range oss {
			c.doSolveOneshot(&oss[idx].OneshotInfo)
		}
		time.Sleep(c.FetchIdleDur)
	}
}

type rangeInfo struct {
	MinRlt int64
	MaxRlt int64
}

func (c *ResolverCtrl) solveGroupRoutine(group string) (err error) {
	base := int64(10)
	if group == "group10" {
		base = 10
	} else if group == "group100" {
		base = 100
	} else {
		return fmt.Errorf("ResolverCtrl.solveGroupRoutine - unknown group %s", group)
	}

	for {
		now := time.Now()

		c.lock.Lock()
		// LastOnchainSec 超过29秒以上的group项，认为已经结算完成
		_, err := c.dbmap.Exec("UPDATE GroupTbl SET Solved=1 WHERE Solved=0 AND LastOnchainSec<? AND GrpType=?", now.Unix()-defaultResolveIntervalSec, group)
		if nil != err {
			log.Printf("c.solveGroupRoutine - c.dbmap.Exec failed : %v", err)
		}
		// 只结算30秒以内未处理过的group项目，先查起止范围
		var ri rangeInfo
		err = c.dbmap.SelectOne(&ri, "SELECT MIN(RltID) AS MinRlt, Max(RltID) AS MaxRlt FROM GroupTbl WHERE Solved=0 AND LastSolveSec<? AND GrpType=?", now.Unix()-defaultResolveIntervalSec, group)
		c.lock.Unlock()
		if nil != err {
			if err.Error() != `sql: Scan error on column index 0: converting driver.Value type <nil> ("<nil>") to a int64: invalid syntax` {
				log.Printf("ResolverCtrl.solveGroupRoutine - c.dbmap.Select range failed : %v", err)
				log.Printf("SELECT MIN(RltID) AS MinRlt, Max(RltID) AS MaxRlt FROM GroupTbl WHERE Solved=0 AND LastSolveSec<%d AND GrpType='%s'", now.Unix()-defaultResolveIntervalSec, group)
			}
			time.Sleep(c.FetchIdleDur * 10)
			continue
		}

		if base*(1+ri.MinRlt/base)-1 > ri.MaxRlt {
			log.Printf("ResolverCtrl.solveGroupRoutine - min %d, max %d, not full, ignore", ri.MinRlt, ri.MaxRlt)
			time.Sleep(c.FetchIdleDur * 10)
			continue
		}

		for rangeBegin := base * (ri.MinRlt / base); rangeBegin+base-1 <= ri.MaxRlt; rangeBegin += base {
			log.Printf("ResolverCtrl.solveGroupRoutine - %s %d", group, rangeBegin)
			c.doSolveGroup(group, rangeBegin)
		}
		time.Sleep(c.FetchIdleDur * 10)
	}
}

func (c *ResolverCtrl) fetchBlockInfo(blkNum uint64) (err error) {
	params := fmt.Sprintf(`{"block_num_or_id": %d}`, blkNum)
	URL := c.RpcURL + "/v1/chain/get_block"
	resp, err := http.Post(URL, "application/json", strings.NewReader(params))

	if nil != err {
		log.Printf("fetchBlockInfo - http.Post(%s) with params %s failed : %v", URL, params, err)
		return err
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		log.Printf("fetchBlockInfo - ioutil.ReadAll failed : %v", err)
		return err
	}

	var rbi respBlkInfo
	if err = json.Unmarshal(buf, &rbi); nil != err {
		log.Printf("fetchBlockInfo - json.Unmarshall failed : %v", err)
		log.Printf("%s", string(buf))
		return err
	}

	// {"code":500,"message":"Internal Service Error","error":{"code":3100002,"name":"unknown_block_exception","what":"Unknown block","details":[]}}
	if rbi.Code != 0 {
		if !strings.Contains(string(buf), `"name":"unknown_block_exception"`) {
			log.Printf("fetchBlockInfo - fetch block %d with http error : %s", blkNum, string(buf))
		}
		return fmt.Errorf("%s", string(buf))
	}

	tmval, err := time.ParseInLocation(parseFormat, rbi.Timestamp, time.UTC)
	if nil != err {
		log.Printf("fetchBlockInfo - time.Parse(%s, %s) failed : %v", parseFormat, rbi.Timestamp, err)
		log.Printf("buf : %s", string(buf))
		log.Printf("rbi : %+v", rbi)
		return err
	}

	c.lock.Lock()
	err = c.dbmap.Insert(&BlockInfoTbl{
		BlockNum:  rbi.BlockNum,
		MicroSec:  tmval.UnixNano() / 1000,
		TmStr:     rbi.Timestamp,
		BlockHash: rbi.ID,
		From:      c.RpcURL,
	})
	c.lock.Unlock()
	if nil != err {
		log.Printf("fetchBlockInfo - Insert failed : %v", err)
	} else {
		now := time.Now()
		log.Printf("fetchBlockInfo - Insert block %d %s @ %s, diff %d second", rbi.BlockNum, rbi.Timestamp, now.Format(time.RFC3339), (now.UnixNano()-tmval.UnixNano())/1000000000)
	}
	return err
}

// curl --request POST --url http://172.17.0.2:8888/v1/chain/get_block -d '{"block_num_or_id":18205030}'
func (c *ResolverCtrl) fetchBlkInfoRoutine() {
	for {
		c.lock.Lock()
		lastNum, err := c.dbmap.SelectInt("SELECT IFNULL(MAX(BlockNum),0) FROM BlockInfoTbl")
		c.lock.Unlock()
		if nil != err {
			log.Printf("fetchBlkInfoRoutine - c.dbmap.SelectInt failed : %v", err)
			continue
		}
		if lastNum < c.FromBlkNum {
			lastNum = c.FromBlkNum
		}

		if lastNum-c.hisHeadBlknum > 180*int64(c.FetchIdleDur)/int64(time.Second) { // 3分钟刷新一次头节点信息
			c.refreshHeadBlk()
		}

		now := time.Now()
		// 计算时间差，避免反复查询
		var tmDiff = now.Sub(c.hisHeadBlkTime)
		blkDiffExpect := int64(tmDiff) / int64(c.BPInterval)

		var nextNum int64
		if c.isForEosforce {
			nextNum = 4 + 4*(lastNum/4) // eosforce 3秒一个块，12秒结算一次
		} else {
			nextNum = 10 + 10*(lastNum/10) // eos 0.5秒一个，10秒结算一次
		}

		if nextNum-c.hisHeadBlknum >= blkDiffExpect-1 {
			time.Sleep(c.FetchIdleDur)
			continue
		}
		//log.Printf("last block : %d, next block %d", lastNum, nextNum)
		err = c.fetchBlockInfo(uint64(nextNum))
		if nil != err {
			if !strings.Contains(err.Error(), `"name":"unknown_block_exception"`) {
				log.Printf("fetchBlkInfoRoutine - fetchRoutine(%d) failed : %v", lastNum, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}
	}
}

func (c *ResolverCtrl) refreshHeadBlk() (err error) {
	URL := c.RpcURL + "/v1/chain/get_info"
	resp, err := http.Post(URL, "application/json", strings.NewReader(""))
	if nil != err {
		log.Printf("ResolverCtrl.refreshHeadBlk - http.Post failed %v", err)
		return err
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		log.Printf("ResolverCtrl.refreshHeadBlk - ioutil.ReadAll failed : %v", err)
		return err
	}

	var rgi respGetInfo
	if err = json.Unmarshal(buf, &rgi); nil != err {
		log.Printf("ResolverCtrl.refreshHeadBlk - json.Unmarshall failed : %v", err)
		log.Printf("%s", string(buf))
		return err
	}

	tmpTm, err := time.ParseInLocation(parseFormat, rgi.HeadBlockTime, time.UTC)
	if nil != err {
		log.Printf("ResolverCtrl.refreshHeadBlk - Parse failed : %v", err)
		return err
	}

	c.hisHeadBlknum = rgi.HeadBlockNum
	c.hisHeadBlkUptime = time.Now()
	c.hisHeadBlkTime = tmpTm

	return nil
}

//curl --request POST --url https://api.eosn.io:443/v1/history/get_actions -d '{"account_name":"guytiobzguge"}'
func (c *ResolverCtrl) fetchActions(fromPos int64) (lastPos int64, err error) {
	URL := c.RpcURL + "/v1/history/get_actions"

	pos := fromPos
	if pos < 0 {
		c.lock.Lock()
		pos, err = c.dbmap.SelectInt("SELECT IFNULL(MAX(AccountActSeq),-1) FROM DiceHistoryTbl")
		c.lock.Unlock()
		if nil != err {
			log.Printf("ResolverCtrl.fetchActions - Select Max(AccountActSeq) failed : %v", err)
			return -1, err
		}

		pos++
	}

	lastPos = pos
	for ; ; pos += 100 {
		lastPos = pos
		params := fmt.Sprintf(`{"account_name":"%s","pos":"%d","offset":"%d"}`,
			c.ContractServant, pos, 100)

		//log.Printf("fetchActions - pos %d begin", pos)
		resp, err := http.Post(URL, "application/json", strings.NewReader(params))
		//log.Printf("fetchActions - pos %d done", pos)

		if nil != err {
			log.Printf("c.fetchActions - pos %d, offset 100 http.PostForm failed : %v", pos, err)
			return -1, err
		}

		buf, err := ioutil.ReadAll(resp.Body)
		if nil != err {
			log.Printf("c.fetchActions ioutil.ReadAll failed : %v", err)
			return -1, err
		}

		var respAct respGetActions
		if err = json.Unmarshal(buf, &respAct); nil != err {
			log.Printf("c.fetchActions - json.Unmarshall failed : %v", err)
			log.Printf("%s", string(buf))
			return -1, err
		}

		if len(respAct.Actions) == 0 {
			log.Printf("c.fetchActions - %s no more actions from %d", c.ContractServant, pos)
			time.Sleep(c.FetchIdleDur)
			return pos, nil
		}
		log.Printf("fetchActions - pos%d，act:%d ", pos, len(respAct.Actions))

		c.lock.Lock()
		trans, err := c.dbmap.Begin()
		if nil != err {
			c.lock.Unlock()
			log.Printf("c.fetchActions - c.dbmap.Begin failed : %v", err)
			return pos, err
		}

		for idx := 0; idx < len(respAct.Actions); idx++ {
			act := &respAct.Actions[idx]
			if act.AccountActionSeq > lastPos {
				lastPos = act.AccountActionSeq
			}
			blockTime, err := time.ParseInLocation("2006-01-02T15:04:05", act.BlockTime, time.UTC)
			if nil != err {
				log.Printf("time.ParseInLocation(\"2006-01-02T15:04:05\", %s, time.UTC) failed : %v",
					act.BlockTime, err)
				continue
			}

			// 只关心 resolveos resolvegrp
			switch act.ActionTrace.Act.Name {
			case "resolveos":
				// "data": {
				//     "osid": 4,
				//     "blknum": 20,
				//     "microsec": "1538018193500000",
				//     "diceval": 67,
				//     "blkid": "000000147743779f0582ad95bddfc575ce8877482708c039fc800926fbc7f47c",
				//     "comment": "{\"bet\":\"10.0000 EOS\",\"celling\":95,\"odds\":1.0425531914893618,\"player\":\"gpmn\",\"res\":\"win\",\"reward\":\"10.4255 EOS\"}"
				// },

				dataMap := act.ActionTrace.Act.Data.((map[string]interface{}))
				osid := uint64(dataMap["osid"].(float64))
				blknum := uint64(dataMap["blknum"].(float64))
				diceval := uint8(dataMap["diceval"].(float64))
				blkid := dataMap["blkid"].(string)
				comment := dataMap["comment"].(string)
				microsec, err := strconv.ParseUint(dataMap["microsec"].(string), 10, 64)
				if nil != err {
					log.Printf("c.fetchActions - strconv.ParseUint %v failed : %v", dataMap["microsec"], err)
					continue
				}

				commentMap := make(map[string]interface{})
				err = json.Unmarshal([]byte(comment), &commentMap)
				if nil != err {
					log.Printf("c.fetchActions - json.Unmarshal comment `%s` failed : %v", comment, err)
					continue
				}

				bet := commentMap["bet"].(string)
				celling := uint8(commentMap["celling"].(float64))
				player := commentMap["player"].(string)
				betRes := commentMap["res"].(string)
				reward := commentMap["reward"].(string)
				betDate := commentMap["betDate"].(string)

				_, err = c.dbmap.Exec("INSERT INTO DiceHistoryTbl (ResolveDate,OsID,BlkNum,BlkID,DiceVal,MicroSec,Celling,Player,Result,Bet,Reward,BetDate,AccountActSeq) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)",
					blockTime, osid, blknum, blkid, diceval, microsec, celling, player, betRes, bet, reward, betDate, act.AccountActionSeq)
				if nil != err {
					log.Printf("c.fetchActions - Insert resolveos failed : %v", err)
				} else {
					log.Printf("c.fetchActions - Insert resolveos %s @ %d - %d", act.ActionTrace.Act.Name, act.AccountActionSeq, osid)
				}
			case "resolvegrp":
				//"data": {
				//   "blknum": 1010,
				//   "microsec": "1538281550500000",
				//   "blkid": "000003f22489ec6558543524f7052a6eefc9bee2f678459822af705e2543baa9",
				//   "diceval": 8,
				//   "forgrp": 10,
				//   "grpbase": 230,
				//   "comment": "{\"dice\":8,\"reward\":\"9.5000 EOS\",\"winner\":238}"
				// },
				dataMap := act.ActionTrace.Act.Data.((map[string]interface{}))
				blknum := uint64(dataMap["blknum"].(float64))
				microsec, err := strconv.ParseUint(dataMap["microsec"].(string), 10, 64)
				if nil != err {
					log.Printf("c.fetchActions - strconv.ParseUint %v failed : %v", dataMap["microsec"], err)
					continue
				}
				blkid := dataMap["blkid"].(string)
				diceval := uint64(dataMap["diceval"].(float64))
				forgrp := int64(dataMap["forgrp"].(float64))
				//grpbase := int64(dataMap["grpbase"].(float64))
				comment := dataMap["comment"].(string)

				grptype := ""
				if forgrp == 10 {
					grptype = "group10"
				} else if forgrp == 100 {
					grptype = "group100"
				} else {
					log.Printf("c.fetchActions - invalid forgrp %d for resolvegrp", forgrp)
					continue
				}

				commentMap := make(map[string]interface{})
				err = json.Unmarshal([]byte(comment), &commentMap)
				if nil != err {
					log.Printf("c.fetchActions - json.Unmarshal comment `%s` failed : %v", comment, err)
					continue
				}
				log.Printf(comment)
				winnerID := uint64(commentMap["winnerID"].(float64))
				winner := commentMap["winner"].(string)
				reward := commentMap["reward"].(string)

				err = c.dbmap.Insert(&GroupHistoryTbl{
					ResolveDate:   blockTime,
					BlkNum:        blknum,
					BlkID:         blkid,
					DiceVal:       diceval,
					MicroSec:      microsec,
					Reward:        reward,
					GrpType:       grptype,
					Winner:        winner,
					WinnerID:      winnerID,
					AccountActSeq: uint64(act.AccountActionSeq),
				})
				if nil != err {
					log.Printf("c.fetchActions - Insert resolvegrp failed : %v", err)
				} else {
					log.Printf("c.fetchActions - Insert %s @ %d - %d %s", act.ActionTrace.Act.Name, act.AccountActionSeq, winnerID, winner)
				}
			default:
				//log.Printf("ignore %s @ %d", act.ActionTrace.Act.Name, act.AccountActionSeq)
			}
		}
		trans.Commit()
		c.lock.Unlock()
		if len(respAct.Actions) < 100 {
			//log.Printf("c.fetchActions - less than 100 one batch, no more")
			time.Sleep(c.FetchIdleDur)
			return lastPos, nil
		}
	}
	return lastPos, nil
}

func (c *ResolverCtrl) fetchActionsRoutine() {
	var err error
	lastPos := int64(-1)
	for {
		lastPos, err = c.fetchActions(lastPos)
		if nil != err {
			lastPos = -1
			time.Sleep(c.FetchIdleDur)
		}
	}
}

func (c *ResolverCtrl) fetchOneshotsRoutine() {
	for {
		err := c.fetchOneshots()
		if nil != err {
			log.Printf("ResolverCtrl.fetchOneshots - c.fetchOneshots failed : %v", err)
			time.Sleep(c.FetchIdleDur)
			continue
		}
	}
}

func (c *ResolverCtrl) fetchGroupRoutine(group string) {
	for {
		err := c.fetchGroup(group)
		if nil != err {
			log.Printf("ResolverCtrl.fetchGroup - %s failed : %v", group, err)
		}
		time.Sleep(c.FetchIdleDur * 5) // group游戏没那么急，多休息下
	}
}

// RunResolve :
func (c *ResolverCtrl) RunResolve() error {
	c.initDB()

	if err := c.refreshHeadBlk(); nil != err {
		log.Printf("main - ctrl.refreshHeadBlk failed : %v", err)
		return err
	}

	go c.fetchBlkInfoRoutine()
	go c.fetchOneshotsRoutine()
	go c.fetchGroupRoutine("group10")
	go c.fetchGroupRoutine("group100")
	go c.solveGroupRoutine("group10")
	go c.solveGroupRoutine("group100")
	c.solveOneshotsRoutine()
	return fmt.Errorf("Unwished exit from ResolverCtrl.Run")
}

// RunFetchActions :
func (c *ResolverCtrl) RunFetchActions() {
	c.initDB()
	c.fetchActionsRoutine()
}

// GetDiceHisTbl :
func (c *ResolverCtrl) GetDiceHisTbl(player string, highBound uint64, limit int) (dht []DiceHistoryTbl, err error) {
	c.lock.Lock()
	if player == "" {
		_, err = c.dbmap.Select(&dht, "SELECT * FROM DiceHistoryTbl WHERE OsID<? ORDER BY OsID DESC LIMIT ?",
			highBound, limit)
	} else {
		_, err = c.dbmap.Select(&dht, "SELECT * FROM DiceHistoryTbl WHERE Player=? AND OsID<? ORDER BY OsID DESC LIMIT ?",
			player, highBound, limit)
	}
	c.lock.Unlock()
	if nil != err {
		log.Printf("ResolverCtrl.GetDiceHisTbl - select high:%d, limit:%d failed %v",
			highBound, limit, err)
		return nil, err
	}
	return dht, nil
}

// GetGrpHisTbl :
func (c *ResolverCtrl) GetGrpHisTbl(group string, limit, offset int64) (grps []GroupHistoryTbl, err error) {
	c.lock.Lock()
	_, err = c.dbmap.Select(&grps, "SELECT * FROM GroupHistoryTbl WHERE GrpType=? ORDER BY AccountActSeq DESC LIMIT ? OFFSET ?", group, limit, offset)
	c.lock.Unlock()
	if nil != err {
		log.Printf("ResolverCtrl.GetGrpHisTbl - failed : %v", err)
		return nil, err
	}
	return grps, nil
}

func calcodds(celling int64) float64 {
	return 100.0 / (float64(celling) - 1.0) * reservedOdds
}
