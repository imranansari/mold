package main

import (
	"testing"
	"time"
)

func Test_LifeCycle_Run(t *testing.T) {
	mc, worker, err := initializeBuild("./testdata/mold3.yml", *dockerURI)
	if err != nil {
		t.Fatal(err)
	}

	lc := NewLifeCycle(worker)
	if err := lc.Run(mc); err != nil {
		t.Fatal(err)
	}
	dw := lc.worker.(*DockerWorker)
	dw.RemoveArtifacts()
}

func Test_LifeCycle_buildless(t *testing.T) {
	mc, worker, err := initializeBuild("./testdata/mold.buildless.yml", *dockerURI)
	if err != nil {
		t.Fatal(err)
	}

	lc := NewLifeCycle(worker)
	if err := lc.Run(mc); err != nil {
		t.Fatal(err)
	}
	dw := lc.worker.(*DockerWorker)
	dw.RemoveArtifacts()
}

func Test_LifeCycle_fail_DueToBadExitCodeInCommands(t *testing.T) {
	mc, _ := readMoldConfig("./testdata/mold.fail.yml")
	mc.RepoName += "-test3"

	worker, err := NewDockerWorker(nil)
	if err != nil {
		t.Fatal(err)
	}
	lc := NewLifeCycle(worker)
	if err := lc.Run(mc); err == nil {
		t.Fatal("should fail")
	}
}

func Test_LifeCycle_Abort(t *testing.T) {
	mc, worker, err := initializeBuild("./testdata/mold3.yml", *dockerURI)
	if err != nil {
		t.Fatal(err)
	}

	mc.RepoName += "-test4"

	lc := NewLifeCycle(worker)
	go func() {
		<-time.After(2750 * time.Millisecond)
		if err := lc.Abort(); err != nil {
			t.Fatal(err)
		}
	}()

	if err := lc.Run(mc); err != nil {
		t.Fatal(err)
	}

}

func Test_LifeCycle_RunTarget(t *testing.T) {
	mc, worker, _ := initializeBuild("./testdata/mold1.yml", *dockerURI)

	mc.RepoName += "-test5"

	lc := NewLifeCycle(worker)
	if err := lc.RunTarget(mc, "artifacts", "d3sw/mold-test"); err != nil {
		t.Fatal(err)
	}

	if err := lc.RunTarget(mc, "build"); err != nil {
		t.Fatal(err)
	}
}

func Test_LifeCycle_BuildContainerCanLocateServiceContainerOnNetwork(t *testing.T) {
	mc, worker, _ := initializeBuild("./testdata/buildlocatesservice.yml", *dockerURI)
	lc := NewLifeCycle(worker)
	if err := lc.Run(mc); err != nil {
		t.Fatal(err)
	}
}

func Test_LifeCycle_multi_artifact(t *testing.T) {
	mc, worker, _ := initializeBuild("./testdata/mold.multi-art.yml", *dockerURI)
	lc := NewLifeCycle(worker)
	if err := lc.Run(mc); err != nil {
		t.Fatal(err)
	}
}
func Test_LifeCycle_duplicate_name(t *testing.T) {
	mc, worker, _ := initializeBuild("./testdata/mold.duplicate.name.yml", *dockerURI)
	lc := NewLifeCycle(worker)
	if err := lc.Run(mc); err != nil {
		t.Fatal(err)
	}
}

func Test_LifeCycle_duplicate_name_fail(t *testing.T) {
	mc, worker, _ := initializeBuild("./testdata/mold.duplicate.name.fail.yml", *dockerURI)
	lc := NewLifeCycle(worker)
	if err := lc.Run(mc); err == nil {
		t.Fatal("should fail with error \"duplicate name\"")
	}
}

func Test_LifeCycle_moldenv(t *testing.T) {
	testMc, _, err := initializeBuild("./testdata/mold.env.yml", "")
	if err != nil {
		t.Fatal(err.Error())
	}
	b := testMc.Build[0]

	envVals, err := b.GetEnvStrings()
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(envVals) != 8 {
		t.Fatalf("environment values: want 8; have %d\n", len(envVals))
	}
}

func Test_LifeCycle_CleanUp(t *testing.T) {
	mc, worker, _ := initializeBuild("./testdata/mold.clean-up.yml", *dockerURI)
	lc := NewLifeCycle(worker)
	if err := lc.Run(mc); err != nil {
		t.Fatal(err)
	}

	if _, err := worker.getImageID("test-image:0.1.0"); err == nil {
		t.Fatal("should fail with \"image not found\"")
	}

	if _, err := worker.getImageID("test-image:0.1.1"); err != nil {
		t.Fatal("should return image id, returned: ", err.Error())
	}
}
