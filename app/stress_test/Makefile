
stress_test:
	gcc -O2 stress_test.c -o stress_test -lm
	

run: stress_test
	sbatch --mem=4G -c 1 --time=00:05:00 --wrap="./stress_test"