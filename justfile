update:
	git pull && ./build/build.sh
	rm -rf /Applications/Oshelly.app
	mv build/Oshelly.app /Applications

open:
	open /Applications/Oshelly.app
