package persistence

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

func PrunePG(pg *sql.DB, pruneDuration time.Duration) (int, int, error) {
	deletedRows := 0
	prunedRows := 0

	//// prune old testinstance output
	//pruneBefore := time.Now().Add(-pruneDuration)
	//result, err = pg.Exec("UPDATE testinstance SET output = '' WHERE enqueued_at < $1", pruneBefore)
	//if err != nil {
	//	return -1, -1, fmt.Errorf("error pruning testinstance output: %v", err)
	//}
	//pruned, err := result.RowsAffected()
	//if err != nil {
	//	return -1, -1, fmt.Errorf("error getting rows affected after pruning testinstance output: %v", err)
	//}
	//prunedRows += int(pruned)

	// delete old testrun entries
	deleteBefore := time.Now().Add(-pruneDuration * 3)

	runDeleteQuery := `
DELETE FROM testrun 
WHERE ref = any (array(SELECT ref FROM testrun WHERE created_at < $1 ORDER BY created_at LIMIT 1000))`
	result, err := pg.Exec(runDeleteQuery, deleteBefore)
	if err != nil {
		return -1, -1, fmt.Errorf("error deleting testrun entries: %v", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return -1, -1, fmt.Errorf("error getting rows affected after deleting testrun entries: %v", err)
	}
	deletedRows += int(deleted)
	log.Printf("Deleted %d testrun entries", deleted)

	// delete test instances that do not have a matching testrun
	result, err = pg.Exec("DELETE FROM testinstance WHERE NOT EXISTS (SELECT FROM testrun WHERE testinstance.testrun_ref = testrun.ref)")
	if err != nil {
		return -1, -1, fmt.Errorf("error deleting testinstance entries: %v", err)
	}
	deleted, err = result.RowsAffected()
	if err != nil {
		return -1, -1, fmt.Errorf("error getting rows affected after deleting testinstance entries: %v", err)
	}
	deletedRows += int(deleted)
	log.Printf("Deleted %d testinstance entries", deleted)

	// delete test upgrades/nodes that do not have a matching testinstance
	result, err = pg.Exec("DELETE FROM clusternode WHERE NOT EXISTS (SELECT FROM testinstance WHERE clusternode.testinstance_id = testinstance.id)")
	if err != nil {
		return -1, -1, fmt.Errorf("error deleting clusternode entries: %v", err)
	}
	deleted, err = result.RowsAffected()
	if err != nil {
		return -1, -1, fmt.Errorf("error getting rows affected after deleting clusternode entries: %v", err)
	}
	deletedRows += int(deleted)
	log.Printf("Deleted %d clusternode entries", deleted)

	result, err = pg.Exec("DELETE FROM testupgrade WHERE NOT EXISTS (SELECT FROM testinstance WHERE testupgrade.id = testinstance.id)")
	if err != nil {
		return -1, -1, fmt.Errorf("error deleting testupgrade entries: %v", err)
	}
	deleted, err = result.RowsAffected()
	if err != nil {
		return -1, -1, fmt.Errorf("error getting rows affected after deleting testupgrade entries: %v", err)
	}
	deletedRows += int(deleted)
	log.Printf("Deleted %d testupgrade entries", deleted)

	return prunedRows, deletedRows, nil
}
