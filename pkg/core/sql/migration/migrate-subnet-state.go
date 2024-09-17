package migration

import (
	"github.com/mlayerprotocol/go-mlayer/internal/sql/models"
	"gorm.io/gorm"
)


 func DropOwnerColumnFromSubnetState(db *gorm.DB) (err error) {
	return db.Migrator().DropColumn(&models.SubnetState{}, "Owner")
 }

 func DropAgentColumnFromSubnetState(db *gorm.DB) (err error) {
	// if db.Migrator().HasColumn(&models.SubnetState{}, "Agent") {
	// 	err = db.Migrator().DropColumn(&models.SubnetState{}, "Agent")
	// }
	return  db.Migrator().DropColumn(&models.SubnetState{}, "Agent")
 }
