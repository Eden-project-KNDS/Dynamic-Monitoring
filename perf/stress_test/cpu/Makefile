
stress_test: stress_test.c
	gcc -O2 stress_test.c -o stress_test -lm

multithreading: multithreading.c
	gcc -O2 multithreading.c -o multithreading -lm

run_stress: stress_test
	sbatch --mem=4G -c 1 --time=00:05:00 --wrap="./stress_test"

run_multi: multithreading
	sbatch --mem=4G -c 1 --time=00:05:00 --wrap="./multithreading"


clean:
	rm *.out stress_test *.err stress_gpu