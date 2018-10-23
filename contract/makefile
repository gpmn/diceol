# eos:
# 	cp foreos.hpp forwhich.hpp
# 	eosiocpp -g diceol.abi diceol.cpp
# 	eosiocpp -o diceol.wast diceol.cpp

eosforce:
	cp foreosforce.hpp forwhich.hpp
	eosiocpp -g diceol.abi diceol.cpp
	eosiocpp -o diceol.wast diceol.cpp

setup: 
	cleos set contract diceonlineos ./ ./diceol.wasm ./diceol.abi -p diceonlineos

clean:
	-rm -rf diceol.wast diceol.abi diceol.wasm
