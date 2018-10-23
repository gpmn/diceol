/*!

   =========================================================
 * Paper Kit 2 - v2.1.0
   =========================================================

 * Product Page: http://www.creative-tim.com/product/paper-kit-2
 * Copyright 2017 Creative Tim (http://www.creative-tim.com)
 * Licensed under MIT (https://github.com/timcreative/paper-kit/blob/master/LICENSE.md)

   =========================================================

 * The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
 */

document.addEventListener('scatterLoaded', scatterExtension => {
    scatter = window.scatter;
    if(scatter == null){
        alert_message("请先安装并配置好scatter!");
        return
    }
    if(navigator.userAgent.includes("MdsApp")){
        window.scatter = null;
    }else{
        delete window.scatter;
    }
    login();
});

var searchVisible = 0;
var transparent = true;

var transparentDemo = true;
var fixedTop = false;

var navbar_initialized = false;

const RESERVED_ODDS = 0.98;
const MAX_CELLING = 95;
const MIN_CELLING = 2;
const MAX_BET = 10;
const MIN_BET = 0.5;

function calc_odds(celling){
    return 100.0/(celling - 1.0) * RESERVED_ODDS;
}

function calc_winchance(celling){
    return (celling - 1.0)/100.0;
}

function timeFormat(tm){
    return tm.getFullYear() + "-" + (tm.getMonth() + 1) + "-" + tm.getDate() + " " +
           tm.getHours() + ":" + tm.getMinutes() + ":" + tm.getSeconds();
}

function update_hints(){
    var celling_slider = document.getElementById('id_slider_celling');
    var bet_slider = document.getElementById('id_slider_bet');
    var celling = celling_slider.noUiSlider.get();
    var bet = bet_slider.noUiSlider.get();

    if(bet < MIN_BET){
        bet = MIN_BET;
    }
    if(bet > MAX_BET){
        bet = MAX_BET;
    }
    if(celling > MAX_CELLING){
        celling = MAX_CELLING;
    }
    if(celling < MIN_CELLING){
        celling = MIN_CELLING;
    }    

    var odds = calc_odds(celling);
    var wc = calc_winchance(celling);
    var reward = bet * odds;

    $("#id_strong_odds").text(odds.toFixed(2));
    $("#id_strong_winchance").text(""+(wc*100).toFixed(0)+"%");
    $("#id_strong_reward").text(reward.toFixed(2));
}

function getURLParams(key)
{
    var query = window.location.search.substring(1);
    var vars = query.split("&");
    for (var i=0;i<vars.length;i++) {
        var pair = vars[i].split("=");
        if(pair[0] == key){return pair[1];}
    }
    return(null);
}

$(document).ready(function(){
    window_width = $(window).width();
    //  Activate the tooltips
    $('[data-toggle="tooltip"]').tooltip();

    if($(".tagsinput").length != 0){
        $(".tagsinput").tagsInput();
    }
    if (window_width >= 768) {
        big_image = $('.page-header[data-parallax="true"]');

        if(big_image.length != 0){
            $(window).on('scroll', pk.checkScrollForPresentationPage);
        }
    }

    if($("#datetimepicker").length != 0){
        $('#datetimepicker').datetimepicker({
            icons: {
                time: "fa fa-clock-o",
                date: "fa fa-calendar",
                up: "fa fa-chevron-up",
                down: "fa fa-chevron-down",
                previous: 'fa fa-chevron-left',
                next: 'fa fa-chevron-right',
                today: 'fa fa-screenshot',
                clear: 'fa fa-trash',
                close: 'fa fa-remove'
            },
            debug: true
        });
    };

    // Activate bootstrap switch
    $('[data-toggle="switch"]').bootstrapSwitch();

    // Navbar color change on scroll
    if($('.navbar[color-on-scroll]').length != 0){
        $(window).on('scroll', pk.checkScrollForTransparentNavbar)
    }

    // Activate tooltips
    $('.btn-tooltip').tooltip();
    $('.label-tooltip').tooltip();

	// Carousel
	$('.carousel').carousel({
        interval: 4000
    });

    $('.form-control').on("focus", function(){
        $(this).parent('.input-group').addClass("input-group-focus");
    }).on("blur", function(){
        $(this).parent(".input-group").removeClass("input-group-focus");
    });

    // Init popovers
    pk.initPopovers();

    // Init Collapse Areas
    pk.initCollapseArea();

    // Init Sliders
    pk.initSliders();

    var invitor = getURLParams("ref")
    if(null != invitor){
        setCookie("invitor",invitor);
    }

    login();
    setInterval(updateBetHistory, 3000);
    setInterval(refresh_group_wrapper, 3000);
    $("#id_welcome_modal").modal('show');
});


$(document).on('click', '.navbar-toggler', function(){
    $toggle = $(this);
    if(pk.misc.navbar_menu_visible == 1) {
        $('html').removeClass('nav-open');
        pk.misc.navbar_menu_visible = 0;
        setTimeout(function(){
            $toggle.removeClass('toggled');
            $('#bodyClick').remove();
        }, 550);
    } else {
        setTimeout(function(){
            $toggle.addClass('toggled');
        }, 580);

        div = '<div id="bodyClick"></div>';
        $(div).appendTo("body").click(function() {
            $('html').removeClass('nav-open');
            pk.misc.navbar_menu_visible = 0;
            $('#bodyClick').remove();
            setTimeout(function(){
                $toggle.removeClass('toggled');
            }, 550);
        });

        $('html').addClass('nav-open');
        pk.misc.navbar_menu_visible = 1;
    }
});

pk = {
    misc:{
        navbar_menu_visible: 0
    },

    checkScrollForPresentationPage: debounce(function(){
        oVal = ($(window).scrollTop() / 3);
        big_image.css({
            'transform':'translate3d(0,' + oVal +'px,0)',
            '-webkit-transform':'translate3d(0,' + oVal +'px,0)',
            '-ms-transform':'translate3d(0,' + oVal +'px,0)',
            '-o-transform':'translate3d(0,' + oVal +'px,0)'
        });
    }, 4),

    checkScrollForTransparentNavbar: debounce(function() {
        if($(document).scrollTop() > $(".navbar").attr("color-on-scroll") ) {
            if(transparent) {
                transparent = false;
                $('.navbar[color-on-scroll]').removeClass('navbar-transparent');
            }
        } else {
            if( !transparent ) {
                transparent = true;
                $('.navbar[color-on-scroll]').addClass('navbar-transparent');
            }
        }
    }, 17),

    initPopovers: function(){
        if($('[data-toggle="popover"]').length != 0){
            $('body').append('<div class="popover-filter"></div>');

            //    Activate Popovers
            $('[data-toggle="popover"]').popover().on('show.bs.popover', function () {
                $('.popover-filter').click(function(){
                    $(this).removeClass('in');
                    $('[data-toggle="popover"]').popover('hide');
                });
                $('.popover-filter').addClass('in');
            }).on('hide.bs.popover', function(){
                $('.popover-filter').removeClass('in');
            });

        }
    },
    initCollapseArea: function(){
        $('[data-toggle="pk-collapse"]').each(function () {
            var thisdiv = $(this).attr("data-target");
            $(thisdiv).addClass("pk-collapse");
        });

        $('[data-toggle="pk-collapse"]').hover(function(){
            var thisdiv = $(this).attr("data-target");
            if(!$(this).hasClass('state-open')){
                $(this).addClass('state-hover');
                $(thisdiv).css({
                    'height':'30px'
                });
            }

        },
                                               function(){
                                                   var thisdiv = $(this).attr("data-target");
                                                   $(this).removeClass('state-hover');

                                                   if(!$(this).hasClass('state-open')){
                                                       $(thisdiv).css({
                                                           'height':'0px'
                                                       });
                                                   }
                                               }).click(function(event){
                                                   event.preventDefault();

                                                   var thisdiv = $(this).attr("data-target");
                                                   var height = $(thisdiv).children('.panel-body').height();

                                                   if($(this).hasClass('state-open')){
                                                       $(thisdiv).css({
                                                           'height':'0px',
                                                       });
                                                       $(this).removeClass('state-open');
                                                   } else {
                                                       $(thisdiv).css({
                                                           'height':height + 30,
                                                       });
                                                       $(this).addClass('state-open');
                                                   }
                                               });
    },
    initSliders: function(){
        // Sliders for demo purpose in refine cards section
        if($('#id_slider_celling').length != 0 ){
            var celling_slider = document.getElementById('id_slider_celling');
            noUiSlider.create(celling_slider, {
            	start: [ 50 ],
                step: 1,
                tooltips: [wNumb({decimals: 0,prefix:"<"})],
            	range: {
            		'min': [  0 ],
            		'max': [ 100 ]
            	}
            });

            celling_slider.noUiSlider.on('change', function (values, handle) {
                console.log("celling changed : ",values, handle);
                val = values[handle];
                if (val < MIN_CELLING) {
                    celling_slider.noUiSlider.set(MIN_CELLING);
                } else if (val > MAX_CELLING) {
                    celling_slider.noUiSlider.set(MAX_CELLING);
                }
                
                update_hints();                
            }); 
        }

        if($('#id_slider_bet').length != 0 ){
            var bet_slider = document.getElementById('id_slider_bet');
            noUiSlider.create(bet_slider, {
            	start: [ 1 ],
                step: 0.1,
                tooltips: [wNumb({decimals: 1,suffix:"EOS"})],
            	range: {
            		'min': [  0 ],
            		'max': [ 10 ]
            	}
            });

            bet_slider.noUiSlider.on('change', function (values, handle) {
                console.log("bet changed : ",values, handle);
                if (values[handle] < 0.5) {
                    bet_slider.noUiSlider.set(0.5);
                }
                update_hints();
            });            
        }

        update_hints();
    },
}

examples = {
    initContactUsMap: function(){
        var myLatlng = new google.maps.LatLng(44.433530, 26.093928);
        var mapOptions = {
            zoom: 14,
            center: myLatlng,
            scrollwheel: false, //we disable de scroll over the map, it is a really annoing when you scroll through page
        }
        var map = new google.maps.Map(document.getElementById("contactUsMap"), mapOptions);

        var marker = new google.maps.Marker({
            position: myLatlng,
            title:"Hello World!"
        });

        // To add the marker to the map, call setMap();
        marker.setMap(map);
    }
}

// Returns a function, that, as long as it continues to be invoked, will not
// be triggered. The function will be called after it stops being called for
// N milliseconds. If `immediate` is passed, trigger the function on the
// leading edge, instead of the trailing.

function debounce(func, wait, immediate) {
	var timeout;
	return function() {
		var context = this, args = arguments;
		clearTimeout(timeout);
		timeout = setTimeout(function() {
			timeout = null;
			if (!immediate) func.apply(context, args);
		}, wait);
		if (immediate && !timeout) func.apply(context, args);
	};
};

function setCookie(cname, cvalue) {
    var d = new Date();
    d.setTime(d.getTime() + (365*24*60*60*1000));
    var expires = "expires="+ d.toUTCString();
    document.cookie = cname + "=" + cvalue + ";" + expires + ";path=/";
}

function getCookie(cname) {
    var name = cname + "=";
    var decodedCookie = decodeURIComponent(document.cookie);
    var ca = decodedCookie.split(';');
    for(var i = 0; i <ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) == ' ') {
            c = c.substring(1);
        }
        if (c.indexOf(name) == 0) {
            return c.substring(name.length, c.length);
        }
    }
    return "";
}

Date.prototype.Format = function(formatString){
	var YYYY,YY,MMMM,MMM,MM,M,DDDD,DDD,DD,D,hhhh,hhh,hh,h,mm,m,ss,s,ampm,AMPM,dMod,th;
	var dateObject = this;
	YY = ((YYYY=dateObject.getFullYear())+"").slice(-2);
	MM = (M=dateObject.getMonth()+1)<10?('0'+M):M;
	MMM = (MMMM=["January","February","March","April","May","June","July","August","September","October","November","December"][M-1]).substring(0,3);
	DD = (D=dateObject.getDate())<10?('0'+D):D;
	DDD = (DDDD=["Sunday","Monday","Tuesday","Wednesday","Thursday","Friday","Saturday"][dateObject.getDay()]).substring(0,3);
	th=(D>=10&&D<=20)?'th':((dMod=D%10)==1)?'st':(dMod==2)?'nd':(dMod==3)?'rd':'th';
	formatString = formatString.replace("#YYYY#",YYYY).replace("#YY#",YY).replace("#MMMM#",MMMM).replace("#MMM#",MMM).replace("#MM#",MM).replace("#M#",M).replace("#DDDD#",DDDD).replace("#DDD#",DDD).replace("#DD#",DD).replace("#D#",D).replace("#th#",th);

	h=(hhh=dateObject.getHours());
	if (h==0) h=24;
	if (h>12) h-=12;
	hh = h<10?('0'+h):h;
    hhhh = hhh<10?('0'+hhh):hhh;
	AMPM=(ampm=hhh<12?'am':'pm').toUpperCase();
	mm=(m=dateObject.getMinutes())<10?('0'+m):m;
	ss=(s=dateObject.getSeconds())<10?('0'+s):s;
	return formatString.replace("#hhhh#",hhhh).replace("#hhh#",hhh).replace("#hh#",hh).replace("#h#",h).replace("#mm#",mm).replace("#m#",m).replace("#ss#",ss).replace("#s#",s).replace("#ampm#",ampm).replace("#AMPM#",AMPM);
}

/* chain apis */
var scatter=null;
var cachedAccount = "";

const DICE_SERVANT = "diceonlineos";
const DCL_TOKEN_HOLDER = "eosio.token";

const FOS_CHAINID = "bd61ae3a031e8ef2f97ee3b0e62776d6d30d4833c8f7c1645c657b149151004b";
const FOS_RPC_HOST = "w1.eosforce.cn"
const FOS_RPC_PORT = 443
const FOS_RPC_PROTO = "https"
const FOS_HTTPRPC = FOS_RPC_PROTO + "://" +  FOS_RPC_HOST + ":" + FOS_RPC_PORT;
const FOS_TRANSFER_CONTRACT = "eosio";

/* 
 * const EOS_CHAINID = "aca376f206b8fc25a6ed44dbdc66547c36c6c33e3a119ffbeaef943642f0e906";
 * const EOS_RPC_HOST = "nodes.get-scatter.com"
 * const EOS_RPC_PORT = 443
 * const EOS_RPC_PROTO = "https"
 */

const EOS_CHAINID = "cf057bbfb72640471fd910bcb67639c22df9f92470936cddc1ade0e2f2e7dc4f";
const EOS_RPC_HOST = "172.17.0.2"
const EOS_RPC_PORT = 8888
const EOS_RPC_PROTO = "http"
const EOS_HTTPRPC = EOS_RPC_PROTO + "://" +  EOS_RPC_HOST + ":" + EOS_RPC_PORT;
const EOS_TRANSFER_CONTRACT = "eosio.token";

var is_mds_wallet = navigator.userAgent.includes("MdsApp");

var is_for_eosforce = (dice_blockchain == "eosforce");/* configured by external */

if(!is_for_eosforce){
    alert("EOS not ready yet!");
}

function get_transfer_contract(){
    return is_for_eosforce ? FOS_TRANSFER_CONTRACT : EOS_TRANSFER_CONTRACT;
}

function get_blockchain(){
    return ((is_mds_wallet && is_for_eosforce) ? 'eosforce' : 'eos');
}

function get_network(){
    /* for eosforce */
    if(is_for_eosforce){
        return {
            protocol:FOS_RPC_PROTO,
            blockchain: get_blockchain(),
            host:FOS_RPC_HOST,
            port:FOS_RPC_PORT,
            chainId: FOS_CHAINID,
        };
    }

    /* for eos */
    return {
        protocol:EOS_RPC_PROTO,
        blockchain: get_blockchain(),
        host:EOS_RPC_HOST,
        port:EOS_RPC_PORT,
        chainId: EOS_CHAINID,
    };
}

function get_query_options(){
    return {
        httpEndpoint   : is_for_eosforce ? FOS_HTTPRPC : EOS_HTTPRPC,
        expireInSeconds: 60,
        broadcast      : true,
        debug          : false,
        sign           : false
    };
}

function scatterAction(contractAccount, action){
    if(scatter == null){
        alert_message("请先安装和配置scatter！");
        return;
    }
    var net = get_network();
    scatter.getIdentity({accounts:[net]}).then(
        identity => {
            //1. get EOS account name
            const account = identity.accounts.find(function(acc){
                console.log(acc);
                return acc.blockchain === get_blockchain();
            });
            console.log("identity info",identity);
            cachedAccount = account.name;
            setCookie("cachedAccount", cachedAccount);

            options = {
                authorization: account.name+'@'+account.authority,
                broadcast: true,
                sign: true
            }
            
            //get EOS instance ,
            const scatterEOS = scatter.eos(net, Eos, options, net.protocol);
            const requiredFields = {
                accounts:[net]
            }
            if(action == undefined){
                console.log("no action param, ignore");
                return;
            }
            
            //exexute contract
            scatterEOS.contract(contractAccount,{requiredFields}).then(contract => {
                action(contract, account);
            }).catch(e => {
                alert_message("Scatter操作失败 : " + e.message);
                console.log("Scatter操作失败 : " + e.message);
            });
        }).catch(
            e => {
                console.log("scatterAction failed : " + e);
                alert_message("Scatter账户初始化失败 : " + e.message);
            }
        );
}

function click_bet(){
    if("" == cachedAccount){
        login();
        return;
    }
    scatterAction(get_transfer_contract(), function(contract, account){
        var celling_slider = document.getElementById('id_slider_celling');
        var bet_slider = document.getElementById('id_slider_bet');
        var celling = parseInt(celling_slider.noUiSlider.get());
        var tmp = bet_slider.noUiSlider.get();
        var bet = parseFloat(tmp).toFixed(4);
        var invitor = getCookie("invitor");
        
        if(celling > MAX_CELLING || celling < MIN_CELLING){
            alert_message("押注标的应该在2到96之间！");
            console("押注标的应该在2到96之间，当前值:" + celling);
            return;
        }

        if(parseFloat(bet) > MAX_BET || parseFloat(bet) < MIN_BET){
            alert_message("押注标的应该在0.5到10之间！");
            console("押注标的应该在0.5到10之间，当前值:" + bet);
            return;
        }
        
        const opts = { authorization:[account.name+'@'+account.authority] };
        var memostr = "oneshot;" + invitor + ";" + celling;
        /* cleos push action eosio transfer '{"from":"eosforce","to":"hello","quantity":"100000.0000 EOS","memo":""}' -p eosforce */
        contract.transfer(account.name, DICE_SERVANT, bet + " EOS", memostr).then(trx => {
            console.log(`Transaction ID: ${trx.transaction_id}`);
        }).catch(e => {
            alert_message("转账失败 ：" + e.message);
            console.error(e);
        });
        refresh_all();
    });
}

function click_grp(grp){
    if("" == cachedAccount){
        login();
        return;
    }
    if(grp == "group10" && grp == "group100"){
        console.log("grp param invalid : " + grp);
        alert_message("grp param invalid : " + grp);
        return;
    }
    
    scatterAction(get_transfer_contract(), function(contract, account){
        var invitor = getCookie("invitor");
        const opts = { authorization:[account.name+'@'+account.authority] };
        var memostr = grp + ";" + invitor + ";";
        contract.transfer(account.name, DICE_SERVANT, "1.0000 EOS", memostr).then(trx => {
            console.log(`Transaction ID: ${trx.transaction_id}`);
        }).catch(e => {
            alert_message("转账失败 ：" + e.message);
            console.error(e);
        });
        refresh_all();
    });
}

function logout(){
    cachedAccount = "";
    setCookie("cachedAccount", "");
    refresh_all();
    
    scatter.forgetIdentity().then((e) => {
        console.log("logout ok");
    }).catch(e=>{
        console.log("logout failed : " + e.message);
    });
}

function login(){
    if(cachedAccount != ""){
        console.log(cachedAccount + " has logged in");
        notify_message(cachedAccount + " has logged in");
        return;
    }
    cachedAccount = getCookie("cachedAccount");
    if(cachedAccount == undefined || cachedAccount == ""){
        cachedAccount = "";
        alert_message("请选择scatter的登录帐号！");
        scatterAction('eosio', function(){
            refresh_all();
        });
    }else{
        refresh_all();
    }
}

/* EOS :: cleos get table eosio.token ACCOUNT accounts 
   {
   "rows": [{
   "balance": "148.0317 EOS"
   }
   ],
   "more": false
   }
 */
function refresh_balance_foreos(){
    /* query EOS */
    Eos(get_query_options()).getTableRows({
        code:EOS_TRANSFER_CONTRACT,
        scope:cachedAccount,
        table:"accounts",
        json:true
    }).then(function(res){
        console.log(res);
        var eos_found = false;
        for(var idx = 0; idx < res.rows.length; idx ++){
            var balance = res.rows[idx].balance;
            var spts = balance.split(" ");
            if(spts[1] == "EOS"){
                eos_found = true;
                $("#id_eos_balance").text(spts[0]);
                $("#id_eos_balance_grp10").text(spts[0]);
                $("#id_eos_balance_grp100").text(spts[0]);
                continue;
            }
        }
        if(!eos_found){
            console.log(cachedAccount + "没有EOS资产");
            $("#id_eos_balance").text("0.0");
            $("#id_eos_balance_grp10").text("0.0");
            $("#id_eos_balance_grp100").text("0.0");
        }
    }).catch(e=>{
        msg = "查询帐号EOS资产'" + cachedAccount + "'失败 : " + e.message;
        console.log(msg);
        alert_message(msg);
    });
    /* query dcl */
    Eos(get_query_options()).getTableRows({
        code:DCL_TOKEN_HOLDER,
        scope:cachedAccount,
        table:"accounts",
        json:true
    }).then(function(res){
        console.log(res);
        var dcl_found = false;
        for(var idx = 0; idx < res.rows.length; idx ++){
            var balance = res.rows[idx].balance;
            var spts = balance.split(" ");
            if(spts[1] == "DCL"){
                dcl_found = true;
                $("#id_dcl_balance").text(spts[0]);
                $("#id_dcl_balance_grp10").text(spts[0]);
                $("#id_dcl_balance_grp100").text(spts[0]);
                continue;
            }
        }
        if(!dcl_found){
            console.log(cachedAccount + "没有DCL资产");
            $("#id_dcl_balance").text("0.0");
            $("#id_dcl_balance_grp10").text("0.0");
            $("#id_dcl_balance_grp100").text("0.0");
        }        
    }).catch(e=>{
        msg = "查询帐DCL资产'" + cachedAccount + "'失败 : " + e.message;
        console.log(msg);
        alert_message(msg);
    });
}

/* EOSFORCE :: cleos get table eosio eosio accounts -k ACCOUNT
   {
   "rows": [{
   "name": "ACCOUNT",
   "available": "4889.9800 EOS"
   }
   ],
   "more": true
   }
 */
function refresh_balance_foreosforce(){    /* query EOSFORCE */
    Eos(get_query_options()).getTableRows({
        code:"eosio",
        scope:"eosio",
        table:"accounts",
        "table_key":cachedAccount,
        json:true
    }).then(function(res){
        console.log(res);
        var eos_found = false;
        for(var idx = 0; idx < res.rows.length; idx ++){
            var balance = res.rows[idx].available;
            var spts = balance.split(" ");
            if(spts[1] == "EOS"){
                eos_found = true;
                $("#id_eos_balance").text(spts[0]);
                $("#id_eos_balance_grp10").text(spts[0]);
                $("#id_eos_balance_grp100").text(spts[0]);
                continue;
            }
        }
        if(!eos_found){
            console.log(cachedAccount + "没有EOS资产");
            $("#id_eos_balance").text("0.0");
            $("#id_eos_balance_grp10").text("0.0");
            $("#id_eos_balance_grp100").text("0.0");
        }
    }).catch(e=>{
        msg = "查询帐号EOS资产'" + cachedAccount + "'失败 : " + e.message;
        console.log(msg);
        alert_message(msg);
    });
    /* query dcl, same as eos :: cleos get table eosio.token diceonlineos accounts */
    Eos(get_query_options()).getTableRows({
        code:DCL_TOKEN_HOLDER,
        scope:cachedAccount,
        table:"accounts",
        json:true
    }).then(function(res){
        console.log(res);
        var dcl_found = false;
        for(var idx = 0; idx < res.rows.length; idx ++){
            var balance = res.rows[idx].balance;
            var spts = balance.split(" ");
            if(spts[1] == "DCL"){
                dcl_found = true;
                $("#id_dcl_balance").text(spts[0]);
                $("#id_dcl_balance_grp10").text(spts[0]);
                $("#id_dcl_balance_grp100").text(spts[0]);
                continue;
            }
        }
        if(!dcl_found){
            console.log(cachedAccount + "没有DCL资产");
            $("#id_dcl_balance").text("0.0");
            $("#id_dcl_balance_grp10").text("0.0");
            $("#id_dcl_balance_grp100").text("0.0");
        }        
    }).catch(e=>{
        msg = "查询帐DCL资产'" + cachedAccount + "'失败 : " + e.message;
        console.log(msg);
        alert_message(msg);
    });
}

function refresh_balance(){
    if("" == cachedAccount){
        $("#id_eos_balance").text("EOS");
        $("#id_eos_balance_grp10").text("EOS");
        $("#id_eos_balance_grp100").text("EOS");
        $("#id_dcl_balance").text("DCL");
        $("#id_dcl_balance_grp10").text("DCL");
        $("#id_dcl_balance_grp100").text("DCL");
        return;
    }

    if(is_for_eosforce){
        refresh_balance_foreosforce();
    }else{
        refresh_balance_foreos();
    }    
}

function refresh_group_wrapper(){
    if($("#group10").hasClass("active")){
        if($("#group10_pool").hasClass("active")){
            refresh_grp_pool("group10");
        }
        if($("#group10_his").hasClass("active")){
            refresh_grp_his("group10");
        }
    }else if($("#group100").hasClass("active")){
        if($("#group100_pool").hasClass("active")){
            refresh_grp_pool("group100");
        }
        if($("#group100_his").hasClass("active")){
            refresh_grp_his("group100");
        }
    }
}

function refresh_grp_his(grp){
    var tblobj;
    if(grp == "group10"){
        tblobj = $("#id_tbody_grp10_history");
    }else if (grp == "group100"){
        tblobj = $("#id_tbody_grp100_history");
    }else{
        alert_message("wrong group : " + grp);
        console.log("wrong group : " + grp);
        return;
    }

    var args="group="+grp+"&limit=20&offset=0&blockchain="+dice_blockchain;
    $.ajax({type:'post',
            url:"/App/GetGrpHisTbl?" + args,
            contentType: "application/json; charset=utf-8",
            error:function(XMLHttpRequest, textStatus, errorThrown){  
                alert_message("查询分组结算情况失败!"+textStatus);
                var msg = "" + textStatus + "." + errorThrown;
                console.log("查询分组结算情况失败 : " + msg);
            },
            success:function(resp){
                if(resp.Result != 0){
                    var msg = "查询分组结算情况错误 : " + resp.Desc;
                    alert_message(msg);
                    console.log(msg);
                    return;
                }
                tblobj.empty();
                console.log(resp);
                for(var idx = 0; idx < resp.Data.length; idx ++){
                    var his = resp.Data[idx];
                    
                    $('<tr class="text-light">'+
                      '<td>' + his.Winner + "@" + his.WinnerID + '</td>' +
                      '<td>' + his.DiceVal + '</td>' +
                      '<td>' + his.Reward.replace(" EOS", "") + '</td>' +
                      '<td class="text_left">' + his.ResolveDate.replace(/T/, " ",-1).replace(/.000000/,"",-1).replace(/.500000/,".5",-1).replace(/Z/,"",-1) + '</td>' + //.replace(/-/g, "",-1).replace(/:/g, "",-1)
                      '</tr>').appendTo(tblobj);
                }
            },
    });
}

function refresh_grp_pool(grp){
    var scope = "";
    var tblobj;
    if(grp == "group10"){
        scope = "grpa";
        tblobj = $("#id_tbody_grp10_players");
    }else if (grp == "group100"){
        scope = "grpb";
        tblobj = $("#id_tbody_grp100_players");
    }else{
        alert_message("wrong group : " + grp);
        console.log("wrong group : " + grp);
        return;
    }
    Eos(get_query_options()).getTableRows({
        code:DICE_SERVANT,
        scope:scope,
        table:"roulette",
        json:true,
        limit:100,
    }).then(function(res){
        tblobj.empty();
        for(var idx = 0; idx < res.rows.length; idx ++){
            var row = res.rows[idx];
            var tmDate = new Date(row.microsec / 1000);
            $('<tr class="text-light">'+
              '<td>' + row.player + "@" + row.rltid + '</td>' +
              '<td class="text_left">' + tmDate.Format("#DD#-#MM#-#YYYY# #hh#:#mm#:#ss#") + '</td>' + 
              '</tr>').appendTo(tblobj);
        }
    }).catch(e=>{
        var msg = "查询多人投注表失败 : " + e.message;
        alert_message(msg);
        console.log(msg);
    });
}

function refresh_account_btn(){
    if("" == cachedAccount){
        $("#id_account").text("---");
    }else{
        $("#id_account").text(cachedAccount);        
    }
}
function refresh_all(){
    refresh_balance();
    refresh_account_btn();
}

function updateBetHistory(){
    var high = 999999999;
    var limit = 20;
    
    var args = "highBound="+high+"&limit="+limit+"&blockchain="+dice_blockchain;

    $.ajax({type:'post',
            url:"/App/GetBetHistory?" + args,
            contentType: "application/json; charset=utf-8",
            error:function(XMLHttpRequest, textStatus, errorThrown){  
                alert_message("查询押注历史失败!"+textStatus);
                var msg = "" + textStatus + "." + errorThrown;
                console.log("查询押注历史失败 : " + msg);
            },
            success:function(resp){
                var tblobj = $("#id_tbody_allhis");
                
                if(resp.Result != 0){
                    var msg = "查询押注历史错误 : " + resp.Desc;
                    alert_message(msg);
                    console.log(msg);
                    return;
                }
                tblobj.empty();                
                for(var idx = 0; idx < resp.Data.length; idx ++){
                    var his = resp.Data[idx];
                    
                    $('<tr class="text-light">'+
                      '<td>' + his.Player + "@" + his.OsID + '</td>' +
                      '<td>' + his.Bet.replace(" EOS", "") + '</td>' +
                      '<td class=' + ((his.DiceVal < his.Celling) ? "text-danger lgfont" : "text-secondary") + '>' + his.Reward.replace(" EOS", "") + '</td>' +
                      '<td>' + his.DiceVal + "/" + his.Celling + '</td>' +
                      '<td class="text_left">' + his.BetDate.replace(/T/, " ",-1).replace(/.000000/,"",-1).replace(/.500000/,".5",-1).replace(/Z/,"",-1) + '</td>' +
                      '</tr>').appendTo(tblobj);
                }
            },
    });
}
