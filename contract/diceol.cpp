// todo::检查变量作用域，不要垮作用域判断、操作成员！
// todo::检查 EOSLIB_SERIALIZE
#include "forwhich.hpp"

#include <string>
#include <vector>
#include <math.h>
#include <iomanip>
#include <eosiolib/eosio.hpp>
#include <eosiolib/asset.hpp>
#include <eosiolib/contract.hpp>
#include <eosiolib/types.h>
#include <eosiolib/currency.hpp>
#include <eosiolib/action.h>
#include <eosiolib/transaction.hpp>
#include <eosiolib/time.hpp>
#include <eosiolib/crypto.h>

#define EOS_SYMBOL S(4,EOS)

using namespace std;

using eosio::name;
using eosio::asset;
using eosio::permission_level;
using eosio::action;
using eosio::print;
using eosio::contract;
using eosio::transaction;
using eosio::time_point_sec;

#define N(X) ::eosio::string_to_name(#X)
#define GROUP_GAME_AMOUNT       10000   // 1 EOS each times for 100 person's group game
#define DEFAULT_MIN_AMOUNT      5000    // at least 0.5 EOS
#define DEFAULT_MAX_AMOUNT      100000  // at most 10 EOS
#define DEFAULT_MIN_CELLING     2       // ratio
#define DEFAULT_MAX_CELLING     95      // ratio
#define RESERVED_ODDS           0.98    // 预留2%的利润给自己
#define DCL_SYMBOL              S(4,DCL)
#define DISTRIBUTION_COMMENT    "Wish DCL could shine your life."
#define INFERRAL_COMMENT        "Thank you for you referral DCL."
#define TYPE_ONESHOT            "oneshot"
#define TYPE_GROUP10            "group10"
#define TYPE_GROUP100           "group100"
#define MAX_PENDING_COUNT       10000


#define DCL_SYMBOL_CONTRACT     N(eosio.token)

#if defined(FOR_EOS)
#warning use 'eosio.token' to transfer
#define EOS_SYMBOL_CONTRACT     N(eosio.token)
#else
#warning use 'eosio' to transfer
#define EOS_SYMBOL_CONTRACT     N(eosio)
#endif

class diceol: public contract {
     struct dlctrl;
     struct sgfinfo;
     
public:
     using contract::contract;
     diceol(account_name self):contract(self),
                               ctrltbl(self, self),
                               ostbl(self, N(oneshot)),
                               grp10tbl(self, N(grpa)),
                               grp100tbl(self, N(grpb)){
     }

     // memo 格式：type;invitor;celling;
     void transfer(account_name code, const eosio::currency::transfer& t){
          // print("transfer from ", eosio::name{t.from}, " -> ", eosio::name{t.to}, ", via code : ", eosio::name{code},
          //       "t.quantity.symbol is :", t.quantity.symbol, ", amount : ", t.quantity.amount, "\n");
          if(t.from == _self){
               print("ignore transfer from myself\n");
               return;
          }
          
          if(t.to != _self){
               print("ignore transfer not to diceol\n");
               return;
          }

          if(t.memo == "internal directly transfer"){
               print("internal directly transfer\n");
               return;
          }
      
#if defined(FOR_EOS)
          eosio_assert(code == N(eosio.token), "invalid contract eos source");
#elif defined(FOR_EOSFORCE)
          eosio_assert(code == N(eosio), "invalid contract eosforce source");
#else
#error must define FOR_EOS OR FOR_EOSFORCE
#endif

          auto ctrlIter = get_ctrl();
          eosio_assert(!ctrlIter->freezed, "contract freezed");
          
          eosio_assert(t.quantity.is_valid(), "invalid quantity");

          eosio_assert(t.quantity.symbol == EOS_SYMBOL,"accept EOS only");
          auto pos0 = t.memo.find(";",0);
          eosio_assert(pos0 != string::npos, "format invalid, no ';' found");
          auto pos1 = t.memo.find(";",pos0+1);
          eosio_assert(pos1 != string::npos, "format invalid, no 2nd ';' found");

          auto strType = t.memo.substr(0,pos0);
          auto strInvitor = t.memo.substr(pos0+1, pos1-pos0 - 1);
          auto strCelling = t.memo.substr(pos1+1);

          trim(strType);
          trim(strInvitor);
          trim(strCelling);
          
          //print("memo is ", t.memo, ", strType is : ", strType, " strInvitor is : ", strInvitor, ", strCelling is : ", strCelling, "\n");

          uint64_t microsecNow = current_time();
          
          if(strType == TYPE_ONESHOT){
               eosio_assert(ctrlIter->os_cnt <= MAX_PENDING_COUNT, "too many pending oneshot bets");
               eosio_assert(t.quantity.amount >= ctrlIter->min_amount, "amount less than valve");
               eosio_assert(t.quantity.amount <= ctrlIter->max_amount, "amount more than valve");
               auto celling = std::stoi(strCelling);
               //print("celling : ", celling, ", max_celling : ", int(ctrlIter->max_celling), "\n");
               eosio_assert(celling <= ctrlIter->max_celling, "celling to high");
               eosio_assert(celling >= ctrlIter->min_celling, "celling to low");

               auto newIter = ostbl.emplace(_self, [&](auto &i){
                         i.osid = ctrlIter->next_osid;
                         i.player = t.from;
                         i.amt = t.quantity.amount;
                         i.celling = celling;
                         i.microsec = microsecNow;
                    });
               //print("newIter -> key : ", newIter->osid);
               
               ctrltbl.modify(ctrlIter, 0, [&](auto &c){
                         c.next_osid ++;
                         c.os_cnt ++;
                    });
          }else if(strType == TYPE_GROUP10){
               eosio_assert(ctrlIter->grp10_cnt <= MAX_PENDING_COUNT, "too many pending group10 bets");
               eosio_assert(t.quantity.amount == GROUP_GAME_AMOUNT, "group game must be 1 eos/bet");
               grp10tbl.emplace(_self, [&](auto &i){
                         i.rltid = ctrlIter->next_grp10id;
                         i.player = t.from;
                         i.microsec = microsecNow;
                    });
               ctrltbl.modify(ctrlIter, 0, [&](auto &c){
                         c.next_grp10id ++;
                         c.grp10_cnt ++;
                    });
          }else if(strType == TYPE_GROUP100){
               eosio_assert(ctrlIter->grp100_cnt <= MAX_PENDING_COUNT, "too many pending group100 bets");
               eosio_assert(t.quantity.amount == GROUP_GAME_AMOUNT, "group game must be 1 eos/bet");
               grp100tbl.emplace(_self, [&](auto &i){
                         i.rltid = ctrlIter->next_grp100id;
                         i.player = t.from;
                         i.microsec = microsecNow;
                    });
               ctrltbl.modify(ctrlIter, 0, [&](auto &c){
                         c.next_grp100id ++;
                         c.grp100_cnt ++;
                    });
          }else{
               eosio_assert(false, "invalid type");
          }          

          // 发送充值奖励token。
          eosio::currency::inline_transfer(_self, t.from,
                                           eosio::extended_asset(eosio::asset{t.quantity.amount/25, DCL_SYMBOL}, DCL_SYMBOL_CONTRACT),
                                           DISTRIBUTION_COMMENT);
          // 发推荐奖励token。
          if(strInvitor != "" && eosio::string_to_name(strInvitor.c_str()) != t.from){
               auto invitor = eosio::string_to_name(strInvitor.c_str());
               eosio_assert(is_account(invitor), "account invalid");
               eosio::currency::inline_transfer(_self, invitor,
                                                eosio::extended_asset(asset{t.quantity.amount/100, DCL_SYMBOL}, DCL_SYMBOL_CONTRACT),
                                                INFERRAL_COMMENT);
          }
     }

     // @abi action
     struct resolveos{
          int64_t osid;        // resolve how many bet this time
          uint64_t blknum;      // block number
          uint64_t microsec;    // block time in unix micro seconds
          uint8_t diceval;      // dice val, blockid ## osid -> sha256 -> sum_of_bytes -> %100
          string blkid;         // blockid
          string comment;
          
          EOSLIB_SERIALIZE(resolveos, (osid)(blknum)(microsec)(diceval)(blkid)(comment));
     };
     
     void do_resolveos(const resolveos& rsv){
          checksum256 hash = {0};
          string hashstr = "";
          int64_t reward = 0;
          eosio_assert(rsv.diceval >= 1 && rsv.diceval <= 100, "diceval invalid");
          require_auth(_self);  // only owner could resolve
          auto ctrlIter = get_ctrl();
          auto ositer = ostbl.find(rsv.osid);
          if(ositer == ostbl.end()){
               eosio_assert(false, (string("no such id in oneshot : ") + std::to_string(rsv.osid)).c_str());
          }

          if(ositer->microsec >= rsv.microsec){
               //print("osid : ", ositer->osid, ", ositer->microsec : ", ositer->microsec, ", rsv.blknum : ", rsv.blknum, ", rsv.microsec :", rsv.microsec,"\n");
               eosio_assert(false ,"not time out, wait for a while");
          }

#if 0
          string buf = rsv.blkid + std::to_string(ositer->osid);
          sha256((char*)buf.c_str(), buf.size(), &hash);
          uint64_t sum = 0;
          for(int idx = 0; idx < sizeof(hash.hash)/sizeof(hash.hash[0]); idx ++){
               sum += (uint8_t)hash.hash[idx];
          }
          sum = 1 + (sum % 100);// [0,99] ==> [1,100]
          eosio_assert(sum == rsv.diceval, "diceval not same as verification");
#endif

          if(rsv.diceval < ositer->celling){ // win
               eosio_assert(ositer->amt >= ctrlIter->min_amount, "amount less than valve");
               eosio_assert(ositer->amt <= ctrlIter->max_amount, "amount more than valve");
               reward = ositer->amt * calcodds(ositer->celling);
          }else{           // lost
               reward = 1;       // if lose, only 0.0001 EOS with comment
          }
          
          // 只要给了comment，就会timeout，啥意思啊，shit
          
          //char comment[256];
          //snprintf(comment, sizeof(comment), "'res':'%s','blknum':'%llu','osid':%llu,'dice':%llu,'celling':%d,'reward':'%.4f EOS'",
          //(reward <= 1) ? "LOST" : "WIN", rsv.blknum, ositer->osid, sum, ositer->celling, reward/10000.0);
          //comment[255] = 0;

          // print((const char*)comment, "\n");

          // string comment = string("{'res':'") + ((reward <= 1) ? "LOST'" : "WIN'")
          //      + ",'dice':" + std::to_string(sum)
          //      + ",'celling':" + std::to_string(ositer->celling)
          // + ",'reward':'" + std::to_string(reward/10000.0) + " EOS'}";

          // string comment = string(reward <= 1 ? "LOST;" : "WIN;")
          //      + std::to_string(rsv.diceval) + ";"
          //      + std::to_string(ositer->celling) + ";"
          //      + std::to_string(reward/10000.0) + " EOS";
          
          //print(rsv.comment);
          
          eosio::currency::inline_transfer(_self, ositer->player,
                                           eosio::extended_asset(asset{reward, EOS_SYMBOL}, EOS_SYMBOL_CONTRACT),
                                           rsv.comment);
          ostbl.erase(ositer);
          ctrltbl.modify(ctrlIter, 0, [&](auto &c){
                    c.os_cnt --;
               });
     }

     // @abi action
     struct resolvegrp{
          uint64_t blknum;      // block number
          uint64_t microsec;    // block time in unix seconds
          string blkid;         // blockid
          int32_t diceval;       // dice val 
          int32_t forgrp;       // 10, for grp10/grpa; 100 for grp100/grpb
          int64_t grpbase;      // 100~200,200~300,etc.
          string comment;
          EOSLIB_SERIALIZE(resolvegrp, (blknum)(microsec)(blkid)(diceval)(forgrp)(grpbase)(comment));
     };
     
     void do_resolvegrp(const resolvegrp& rsv){
          require_auth(_self);  // only owner could resolve
          auto ctrlIter = get_ctrl();
          roulette_index_t* tblptr;
          int64_t bonus;
          
          if(rsv.forgrp == 10){
               eosio_assert((rsv.grpbase % 10) == 0 && ctrlIter->next_grp10id >= rsv.grpbase + 10, "grp base not 10 times or not full");
               tblptr = &grp10tbl;
               bonus = 10 * 10000 * 0.95;
          }else if (rsv.forgrp == 100){
               eosio_assert((rsv.grpbase % 100) == 0 && ctrlIter->next_grp100id >= rsv.grpbase + 100, "grp base not 100 times or not full");
               tblptr = &grp100tbl;
               bonus = 100 * 10000 * 0.95;
          }else{
               eosio_assert(false, "forgrp invalid");
          }

          auto lastIter = tblptr->find((rsv.forgrp == 10) ? rsv.grpbase + 9 : rsv.grpbase + 99);
          eosio_assert(lastIter != tblptr->end(), "group not full?");
          //print("id:",lastIter->rltid, ",microsec:",lastIter->microsec,"rsv.microsec:",rsv.microsec,"\n");
          eosio_assert(lastIter->microsec < rsv.microsec, "not time out, wait for a while");

          uint64_t sum = 0; // sum of hash, then mod 100 or 10
#if 0
          checksum256 hash = {0};
          sha256((char*)rsv.blkid.c_str(), rsv.blkid.size(), &hash);
          for(int idx = 0; idx < sizeof(hash.hash)/sizeof(hash.hash[0]); idx ++){
               sum += hash.hash[idx];
          }
          //print("sum : ", sum, "-> calc dice : ", (sum % rsv.forgrp));
          sum = sum % rsv.forgrp;
          eosio_assert(sum == rsv.diceval, "calc diceval not same as input");
#endif
          
          auto iter = tblptr->find(rsv.grpbase + rsv.diceval);
          eosio_assert(iter != tblptr->end(), (string("no such grp winner id : ") + std::to_string(rsv.grpbase + sum)).c_str());

          eosio::currency::inline_transfer(_self, iter->player,
                                           eosio::extended_asset(asset{bonus, EOS_SYMBOL}, EOS_SYMBOL_CONTRACT),
                                           rsv.comment);

          for(auto idx = rsv.grpbase; idx < rsv.grpbase + rsv.forgrp; idx ++){
               iter = tblptr->find(idx);
               if(iter == tblptr->end()){
                    eosio_assert(false, (string("can not find item id @ ") + std::to_string(idx)).c_str());
               }
               tblptr->erase(iter);
          }
          ctrltbl.modify(ctrlIter, 0, [&](auto &c){
                    if(rsv.forgrp == 10){
                         c.grp10_cnt -=10;
                    }else if(rsv.forgrp == 100){
                         c.grp100_cnt -= 100;
                    }
               });
     }
     
     void apply(account_name code, account_name action) {
          if(code != _self){   // notification handlers, diceol is not action target, just be notified
               switch(action){
               case N(transfer):
                    transfer(code, eosio::unpack_action_data<eosio::currency::transfer>());
                    return;
               default:
                    print("ignore invalid notification source : ", eosio::name{code}, ", action : ", eosio::name{action},"\n");
                    return;
               }
          }

          // case we are the action target
          switch(action){
          case N(resolveos):
               do_resolveos(eosio::unpack_action_data<resolveos>());
               return;
          case N(resolvegrp):
               do_resolvegrp(eosio::unpack_action_data<resolvegrp>());
               return;
          case N(transfer): // inline transfer from us to others, ignore
               return;
          default:
               print("ignore unknown action : ", eosio::name{action}, "\n");
               eosio_assert(false, "unknown action");
          }
     }
     
private:
     // @abi table
     struct dlctrl{
          uint64_t id;
          uint64_t next_osid;
          uint64_t next_grp10id;
          uint64_t next_grp100id;
          int64_t os_cnt;
          int64_t grp10_cnt;
          int64_t grp100_cnt;
          bool freezed;
          int8_t min_celling, max_celling;
          int64_t min_amount, max_amount;

          int64_t rsv0,rsv1,rsv2;
          uint64_t primary_key() const {return id;}
          
          EOSLIB_SERIALIZE(dlctrl, (id)(next_osid)(next_grp10id)(next_grp100id)(os_cnt)(grp10_cnt)(grp100_cnt)(freezed)(min_celling)(max_celling)(min_amount)(max_amount)(rsv0)(rsv1)(rsv2))
     };
     
     typedef eosio::multi_index<N(dlctrl), dlctrl> dlctrl_index_t;
     dlctrl_index_t ctrltbl;

     eosio::multi_index<N(dlctrl), dlctrl>::const_iterator get_ctrl(){
          auto ctrlIter = ctrltbl.begin();
          if(ctrlIter == ctrltbl.end()) {
               ctrlIter = ctrltbl.emplace(_self, [&](auto& a){
                         a.id = 0;
                         a.next_osid = 0;
                         a.next_grp10id = 0;
                         a.next_grp100id = 0;
                         a.os_cnt = 0;
                         a.grp10_cnt = 0;
                         a.grp100_cnt = 0;
                         a.min_amount = DEFAULT_MIN_AMOUNT;
                         a.max_amount = DEFAULT_MAX_AMOUNT;
                         a.rsv0 = 0;
                         a.rsv1 = 0;
                         a.rsv2 = 0;
                         a.min_celling = DEFAULT_MIN_CELLING;
                         a.max_celling = DEFAULT_MAX_CELLING;
                         a.freezed = false;
                    });
          }
          eosio_assert(ctrlIter != ctrltbl.end(), "iterator to the end of ctrltbl");
          return ctrlIter;
     }     

     //@abi table
     struct oneshot{            // 一次性的投注方式
          uint64_t osid;        // id
          account_name player;  // who is bet
          uint64_t amt;         // bet amount
          uint8_t celling;      // will win, if dice < celling
          uint64_t microsec;    // bet at when in number;
          uint64_t primary_key() const {return osid;}
          EOSLIB_SERIALIZE(oneshot, (osid)(player)(amt)(celling)(microsec))
     };
     typedef eosio::multi_index<N(oneshot), oneshot> oneshot_index_t;
     oneshot_index_t ostbl;

     //@abi table
     struct roulette{            // 轮盘赌
          uint64_t rltid;        // id
          account_name player;   // who is bet
          uint64_t microsec;     // bet at when in number;
          uint64_t primary_key() const {return rltid;}
          EOSLIB_SERIALIZE(roulette, (rltid)(player)(microsec))
     };
     typedef eosio::multi_index<N(roulette), roulette> roulette_index_t;
     roulette_index_t grp10tbl;
     roulette_index_t grp100tbl;
     
     string& bin2str(uint8_t array[], int len, string& res){
          res = "";
          for(int idx=0; idx<len; idx++){
               uint8_t val = array[idx];
               uint8_t high = val >> 4;
               uint8_t low = val & 0xf;
               res += (high<10) ? ('0' + high) : ('A' + high - 10);
               res += (low<10) ? ('0' + low) : ('A' + low - 10);
          }
          return res;
     }

     inline float calcodds(int celling){
          return 100.0/(float(celling) - 1.0) * RESERVED_ODDS;
     }

     inline string& trim(string &s) {
          if (s.empty()) {
               return s;
          }

          s.erase(0,s.find_first_not_of(" "));
          s.erase(s.find_last_not_of(" ") + 1);
          return s;
     }
};

extern "C" {
     void apply(uint64_t receiver, uint64_t code, uint64_t action) {
          diceol s(receiver);
          s.apply(code, action);
     }
}
