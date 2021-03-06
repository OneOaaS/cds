package notification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// SendPipeline sends a pipeline notification at end of build
func SendPipeline(db database.QueryExecuter, pb *sdk.PipelineBuild, event sdk.NotifEventType, status sdk.Status, previous *sdk.PipelineBuild) {
	log.Debug("notification.SendPipelineBuild> pb:%d event:%s status:%s", pb.ID, event, status)

	//Send PipelineBuildNotif
	n := &sdk.Notif{
		DateNotif: time.Now().Unix(),
		Build:     pb,
		Event:     event,
		Status:    status,
		NotifType: sdk.PipelineBuildNotif,
	}

	go post(n)

	//Send UserNotif
	//Load notif
	userNotifs, err := LoadUserNotificationSettings(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID)
	if err != nil {
		log.Critical("notification.SendPipelineBuild> error while loading user notification settings : %s", err)
		return
	}
	if userNotifs == nil {
		log.Debug("notification.SendPipelineBuild> no user notification on pipeline %d, app %d, env %d", pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID)
		return
	}

	//Compute notification
	params := map[string]string{}
	for _, p := range pb.Parameters {
		params[p.Name] = p.Value
	}
	params["cds.status"] = n.Status.String()
	//Set PipelineBuild UI URL
	params["cds.buildURL"] = fmt.Sprintf("%s/#/project/%s/application/%s/pipeline/%s/build/%d?env=%s&tab=detail", baseURL, pb.Pipeline.ProjectKey, pb.Application.Name, pb.Pipeline.Name, pb.BuildNumber, pb.Environment.Name)
	//find author (triggeredBy user or changes author)
	if pb.Trigger.TriggeredBy != nil {
		params["cds.author"] = pb.Trigger.TriggeredBy.Username
	} else if pb.Trigger.VCSChangesAuthor != "" {
		params["cds.author"] = pb.Trigger.VCSChangesAuthor
	}

	for t, notif := range userNotifs.Notifications {
		if ShouldSendUserNotification(notif, pb, previous) {
			switch t {
			case sdk.JabberUserNotification:
				jn, ok := notif.(*sdk.JabberEmailUserNotificationSettings)
				if !ok {
					log.Critical("notification.SendPipelineBuild> cannot deal with %s", notif)
				}
				//Get recipents from groups
				if jn.SendToGroups {
					u, err := permission.ApplicationPipelineEnvironmentUsers(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, permission.PermissionRead)
					if err != nil {
						log.Critical("notification[Jabber].SendPipelineBuild> error while loading permission :%s", err.Error())
						return
					}
					for i := range u {
						jn.Recipients = append(jn.Recipients, u[i].Username)
					}
				}
				//Get recipents from groups
				if jn.SendToGroups {
					u, err := permission.ApplicationPipelineEnvironmentUsers(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, permission.PermissionRead)
					if err != nil {
						log.Critical("notification[Jabber].SendPipelineBuild> error while loading permission :%s", err.Error())
						return
					}
					for i := range u {
						jn.Recipients = append(jn.Recipients, u[i].Username)
					}
				}
				if jn.SendToAuthor {
					//find author (triggeredBy user or changes author)
					if pb.Trigger.TriggeredBy != nil {
						jn.Recipients = append(jn.Recipients, pb.Trigger.TriggeredBy.Username)
					} else if pb.Trigger.VCSChangesAuthor != "" {
						jn.Recipients = append(jn.Recipients, pb.Trigger.VCSChangesAuthor)
					}
				}
				//Finally deduplicate everyone
				removeDuplicates(&jn.Recipients)

				notif, err := jabberEmailNotif(pb, jn, params)
				if err != nil {
					log.Critical("notification[Jabber].SendPipelineBuild> error getting jabber/email notification %s", err.Error())
				}

				log.Notice("Notification[Jabber]> Send jabber notif '%s'", notif.Title)
				go post(&notif)

			case sdk.EmailUserNotification:
				jn, ok := notif.(*sdk.JabberEmailUserNotificationSettings)
				if !ok {
					log.Critical("notification.SendPipelineBuild> cannot deal with %s", notif)
				}
				//Get recipents from groups
				if jn.SendToGroups {
					u, err := permission.ApplicationPipelineEnvironmentUsers(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, permission.PermissionRead)
					if err != nil {
						log.Critical("notification[Email].SendPipelineBuild> error while loading permission :%s", err.Error())
						return
					}
					for i := range u {
						jn.Recipients = append(jn.Recipients, u[i].Email)
					}
				}
				if jn.SendToAuthor {
					var username string
					if pb.Trigger.TriggeredBy != nil {
						username = pb.Trigger.TriggeredBy.Username
					} else if pb.Trigger.VCSChangesAuthor != "" {
						username = pb.Trigger.VCSChangesAuthor
					}
					if username != "" {
						u, err := pipelineInitiator(db, username)
						if err != nil {
							log.Warning("notification[Email].SendPipelineBuild> Cannot load author %s: %s", username, err)
							continue
						}
						jn.Recipients = append(jn.Recipients, u.Email)
					}
				}
				//Finally deduplicate everyone
				removeDuplicates(&jn.Recipients)

				notif, err := jabberEmailNotif(pb, jn, params)
				if err != nil {
					log.Critical("notification[Email].SendPipelineBuild> error getting jabber/email notification %s", err.Error())
				}

				log.Notice("Notification[Email]> Send mail notif '%s'", notif.Title)
				go SendMailNotif(&notif)
			}
		}
	}
}

func removeDuplicates(xs *[]string) {
	found := make(map[string]bool)
	j := 0
	for i, x := range *xs {
		if !found[x] {
			found[x] = true
			(*xs)[j] = (*xs)[i]
			j++
		}
	}
	*xs = (*xs)[:j]
}

//ShouldSendUserNotification check if user notification has to be sent
func ShouldSendUserNotification(notif sdk.UserNotificationSettings, current *sdk.PipelineBuild, previous *sdk.PipelineBuild) bool {
	var check = func(s sdk.UserNotificationEventType) bool {
		switch s {
		case sdk.UserNotificationAlways:
			return true
		case sdk.UserNotificationNever:
			return false
		case sdk.UserNotificationChange:
			if previous == nil {
				return true
			}
			return current.Status != previous.Status
		}
		return false
	}
	switch current.Status {
	case sdk.StatusSuccess:
		if check(notif.Success()) {
			return true
		}
	case sdk.StatusFail:
		if check(notif.Failure()) {
			return true
		}
	case sdk.StatusBuilding:
		return notif.Start()
	}

	return false
}

func jabberEmailNotif(pb *sdk.PipelineBuild, notif *sdk.JabberEmailUserNotificationSettings, params map[string]string) (sdk.Notif, error) {
	title := notif.Template.Subject
	message := notif.Template.Body
	for k, value := range params {
		key := "{{." + k + "}}"
		title = strings.Replace(title, key, value, -1)
		message = strings.Replace(message, key, value, -1)
	}

	n := sdk.Notif{
		DateNotif:   time.Now().Unix(),
		Status:      pb.Status,
		NotifType:   sdk.UserNotif,
		Destination: "jabber",
		Title:       title,
		Message:     message,
	}
	for _, r := range notif.Recipients {
		n.Recipients = append(n.Recipients, r)
	}

	log.Debug("notification.jabberEmailNotif> send [%s]%s to (%s)", n.Title, n.Message, strings.Join(n.Recipients, ","))

	return n, nil
}

//UserNotificationInput is a way to parse notification
type UserNotificationInput struct {
	Notifications         map[string]interface{} `json:"notifications"`
	ApplicationPipelineID int64                  `json:"application_pipeline_id"`
	Environment           sdk.Environment        `json:"environment"`
	Pipeline              sdk.Pipeline           `json:"pipeline"`
}

//ParseUserNotification transform jsons to UserNotificationSettings map
func ParseUserNotification(body []byte) (*sdk.UserNotification, error) {

	var input = &UserNotificationInput{}
	if err := json.Unmarshal(body, &input); err != nil {
		return nil, err
	}
	settingsBody, err := json.Marshal(input.Notifications)
	if err != nil {
		return nil, err
	}

	var notif1 = &sdk.UserNotification{
		ApplicationPipelineID: input.ApplicationPipelineID,
		Environment:           input.Environment,
		Pipeline:              input.Pipeline,
	}

	notif1.Notifications, err = ParseUserNotificationSettings(settingsBody)
	return notif1, nil
}

//ParseUserNotificationSettings transforms json to UserNotificationSettings map
func ParseUserNotificationSettings(settings []byte) (map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings, error) {
	mapSettings := map[string]interface{}{}
	if err := json.Unmarshal(settings, &mapSettings); err != nil {
		return nil, err
	}

	notifications := map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{}

	for k, v := range mapSettings {
		switch k {
		case string(sdk.EmailUserNotification), string(sdk.JabberUserNotification):
			if v != nil {
				var x sdk.JabberEmailUserNotificationSettings
				tmp, err := json.Marshal(v)
				if err != nil {
					log.Warning("ParseUserNotificationSettings> unable to parse JabberEmailUserNotificationSettings : %s", err)
					return nil, sdk.ErrParseUserNotification
				}
				if err := json.Unmarshal(tmp, &x); err != nil {
					log.Warning("ParseUserNotificationSettings> unable to parse JabberEmailUserNotificationSettings : %s", err)
					return nil, sdk.ErrParseUserNotification
				}
				notifications[sdk.UserNotificationSettingsType(k)] = &x
			}
		case string(sdk.TATUserNotification):
			if v != nil {
				var x sdk.TATUserNotificationSettings
				tmp, err := json.Marshal(v)
				if err != nil {
					log.Warning("ParseUserNotificationSettings> unable to parse TATUserNotificationSettings : %s", err)
					return nil, sdk.ErrParseUserNotification
				}
				if err := json.Unmarshal(tmp, &x); err != nil {
					log.Warning("ParseUserNotificationSettings> unable to parse TATUserNotificationSettings : %s", err)
					return nil, sdk.ErrParseUserNotification
				}
				notifications[sdk.UserNotificationSettingsType(k)] = &x
			}
		default:
			log.Critical("ParseUserNotificationSettings> unsupported %s", k)
			return nil, sdk.ErrNotSupportedUserNotification
		}
	}

	return notifications, nil
}

//LoadAllUserNotificationSettings load data from application_pipeline_notif
func LoadAllUserNotificationSettings(db database.Querier, appID int64) ([]sdk.UserNotification, error) {
	n := []sdk.UserNotification{}
	query := `
		SELECT 	application_pipeline_id, environment_id, settings, pipeline.id, pipeline.name, environment.name
		FROM  	application_pipeline_notif
		JOIN 	application_pipeline ON application_pipeline.id = application_pipeline_notif.application_pipeline_id
		JOIN 	pipeline ON pipeline.id = application_pipeline.pipeline_id
		JOIN 	environment ON environment.id = environment_id
		WHERE 	application_pipeline.application_id = $1
		ORDER BY pipeline.name
	`

	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var un sdk.UserNotification
		var settings string
		err := rows.Scan(&un.ApplicationPipelineID, &un.Environment.ID, &settings, &un.Pipeline.ID, &un.Pipeline.Name, &un.Environment.Name)
		if err != nil {
			return nil, err
		}
		un.Notifications, err = ParseUserNotificationSettings([]byte(settings))
		if err != nil {
			return nil, err
		}
		n = append(n, un)
	}

	return n, nil
}

//LoadUserNotificationSettings load data from application_pipeline_notif
func LoadUserNotificationSettings(db database.Querier, appID, pipID, envID int64) (*sdk.UserNotification, error) {
	var n = &sdk.UserNotification{}
	var settings string
	query := `
		SELECT 	application_pipeline_id, environment_id, settings, pipeline.id, pipeline.name, environment.name
		FROM  	application_pipeline_notif
		JOIN 	application_pipeline ON application_pipeline.id = application_pipeline_notif.application_pipeline_id
		JOIN 	pipeline ON pipeline.id = application_pipeline.pipeline_id
		JOIN 	environment ON environment.id = environment_id
		WHERE 	application_pipeline.application_id = $1
		AND		application_pipeline.pipeline_id = $2
		AND 	application_pipeline_notif.environment_id = $3
		ORDER BY pipeline.name
	`

	if err := db.QueryRow(query, appID, pipID, envID).Scan(&n.ApplicationPipelineID, &n.Environment.ID, &settings,
		&n.Pipeline.ID, &n.Pipeline.Name, &n.Environment.Name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Warning("notification.LoadUserNotificationSettings>1> %s", err)
		return nil, err
	}

	var err error
	n.Notifications, err = ParseUserNotificationSettings([]byte(settings))
	if err != nil {
		log.Warning("notification.LoadUserNotificationSettings>2> %s", err)
		return nil, err
	}

	return n, nil
}

// DeleteNotification Delete a notifications for the given application/pipeline/environment
func DeleteNotification(db database.QueryExecuter, appID, pipID, envID int64) error {
	query := `
		DELETE FROM application_pipeline_notif
		USING 	application_pipeline, application, pipeline
		WHERE application_pipeline.id = application_pipeline_notif.application_pipeline_id
		AND application_pipeline.application_id = $1
		AND application_pipeline.pipeline_id = $2
		AND application_pipeline_notif.environment_id = $3
	`
	_, err := db.Exec(query, appID, pipID, envID)
	return err
}

//InsertOrUpdateUserNotificationSettings insert or update value in application_pipeline_notif
func InsertOrUpdateUserNotificationSettings(db database.QueryExecuter, appID, pipID, envID int64, notif *sdk.UserNotification) error {
	query := `
		SELECT 	count(1) 
		FROM  	application_pipeline_notif
		JOIN 	application_pipeline ON application_pipeline.id = application_pipeline_notif.application_pipeline_id
		WHERE 	application_pipeline.application_id = $1
		AND		application_pipeline.pipeline_id = $2
		AND 	application_pipeline_notif.environment_id = $3
	`

	var nb int
	if err := db.QueryRow(query, appID, pipID, envID).Scan(&nb); err != nil {
		log.Critical("notification.InsertOrUpdateUserNotificationSettings> Error counting application_pipeline_notif %d %d %d : %s", appID, pipID, envID, err)
		return err
	}
	var appPipelineID int64
	if nb == 0 {
		query = `
			INSERT INTO application_pipeline_notif (application_pipeline_id, environment_id) 
			VALUES (
				(
				SELECT 	application_pipeline.id
				FROM 	application_pipeline
				WHERE 	application_pipeline.application_id = $1
				AND		application_pipeline.pipeline_id = $2
				),$3
			) 
			RETURNING application_pipeline_id
		`
		if err := db.QueryRow(query, appID, pipID, envID).Scan(&appPipelineID); err != nil {
			log.Critical("notification.InsertOrUpdateUserNotificationSettings> Error inserting application_pipeline_notif %d %d %d : %s", appID, pipID, envID, err)
			return err
		}
	}

	if appPipelineID != 0 {
		notif.ApplicationPipelineID = appPipelineID
		notif.Environment.ID = envID
	}

	bytes, err := json.Marshal(notif.Notifications)
	if err != nil {
		log.Critical("notification.InsertOrUpdateUserNotificationSettings> Error marshalling notifications settings : %s", err)
		return err
	}

	//Update settings
	query = `
		UPDATE 	application_pipeline_notif SET settings = $4 
		FROM 	application_pipeline
		WHERE 	application_pipeline.application_id = $1
		AND		application_pipeline.pipeline_id = $2
		AND 	application_pipeline_notif.environment_id = $3
		AND 	application_pipeline.id = application_pipeline_notif.application_pipeline_id
	`
	res, err := db.Exec(query, appID, pipID, envID, string(bytes))
	if err != nil {
		log.Critical("notification.InsertOrUpdateUserNotificationSettings> Error updating notifications settings %d %d %d : %s", appID, pipID, envID, err)
		return err
	}
	if i, _ := res.RowsAffected(); i != 1 {
		log.Critical("notification.InsertOrUpdateUserNotificationSettings> Error updating notifications settings %d %d %d : %d rows updated", appID, pipID, envID, i)
		return fmt.Errorf("Error updating notifications settings %d %d %d : %d rows updated", appID, pipID, envID, i)
	}

	return nil
}

func pipelineInitiator(db database.Querier, username string) (*sdk.User, error) {
	query := `
		SELECT data FROM "user"
		WHERE "user".username = $1
	`
	var data string
	err := db.QueryRow(query, username).Scan(&data)
	if err != nil {
		return nil, err
	}

	// Load user
	u, err := sdk.NewUser(username).FromJSON([]byte(data))
	return u, err
}
