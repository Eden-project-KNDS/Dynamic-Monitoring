
stress_test: stress_test.c
	gcc -O2 stress_test.c -o stress_test -lm


run_stress: stress_test
	sbatch --mem=4G -c 1 --time=00:05:00 --wrap="./stress_test"



clean:
	rm *.out stress_test *.err stress_gpu