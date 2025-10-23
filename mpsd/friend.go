package mpsd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/df-mc/go-xsapi"
	"github.com/df-mc/go-xsapi/internal"
	"github.com/google/uuid"
)

const (
	MinecraftTitleID         = "1739947436"
	MinecraftServiceConfigID = "4fc10100-5f7a-4470-899b-280835760c07"
)

// FriendMinecraftStatus represents a friend's Minecraft status
type FriendMinecraftStatus struct {
	Gamertag string `json:"gamertag"`
	XUID     string `json:"xuid"`
	TitleID  string `json:"titleId"`
}

// JoinableFriends returns only friends who are in joinable Minecraft worlds
func (conf PublishConfig) JoinableFriends(ctx context.Context, src xsapi.TokenSource) ([]FriendMinecraftStatus, error) {
	if conf.Client == nil {
		conf.Client = &http.Client{}
	}
	internal.SetTransport(conf.Client, src)

	filter := ActivityFilter{
		Client:      conf.Client,
		SocialGroup: SocialGroupPeople,
	}

	serviceConfigID, err := uuid.Parse(MinecraftServiceConfigID)
	if err != nil {
		return nil, fmt.Errorf("parse service config ID: %w", err)
	}

	activities, err := filter.Search(src, serviceConfigID)
	if err != nil {
		return nil, fmt.Errorf("search activities: %w", err)
	}

	tok, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("obtain token: %w", err)
	}
	currentUserXUID := tok.DisplayClaims().XUID

	joinableFriends := make([]FriendMinecraftStatus, 0)
	for _, activity := range activities {
		if activity.TitleID == MinecraftTitleID && activity.OwnerXUID != currentUserXUID {
			canJoin := activity.RelatedInfo != nil && !activity.RelatedInfo.Closed

			if canJoin {
				gamertag, err := conf.getGamertagFromXUID(ctx, activity.OwnerXUID)
				if err != nil {
					gamertag = activity.OwnerXUID
				}

				status := FriendMinecraftStatus{
					Gamertag: gamertag,
					XUID:     activity.OwnerXUID,
					TitleID:  activity.TitleID,
				}
				joinableFriends = append(joinableFriends, status)
			}
		}
	}

	return joinableFriends, nil
}

// OnlineFriends returns all friends who are currently online
func (conf PublishConfig) OnlineFriends(ctx context.Context, src xsapi.TokenSource) ([]FriendMinecraftStatus, error) {
	if conf.Client == nil {
		conf.Client = &http.Client{}
	}
	internal.SetTransport(conf.Client, src)

	filter := ActivityFilter{
		Client:      conf.Client,
		SocialGroup: SocialGroupPeople,
	}

	serviceConfigID, err := uuid.Parse(MinecraftServiceConfigID)
	if err != nil {
		return nil, fmt.Errorf("parse service config ID: %w", err)
	}

	activities, err := filter.Search(src, serviceConfigID)
	if err != nil {
		return nil, fmt.Errorf("search activities: %w", err)
	}

	tok, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("obtain token: %w", err)
	}
	currentUserXUID := tok.DisplayClaims().XUID

	onlineFriends := make([]FriendMinecraftStatus, 0)
	for _, activity := range activities {
		if activity.TitleID == MinecraftTitleID && activity.OwnerXUID != currentUserXUID {
			gamertag, err := conf.getGamertagFromXUID(ctx, activity.OwnerXUID)
			if err != nil {
				gamertag = activity.OwnerXUID
			}

			status := FriendMinecraftStatus{
				Gamertag: gamertag,
				XUID:     activity.OwnerXUID,
				TitleID:  activity.TitleID,
			}
			onlineFriends = append(onlineFriends, status)
		}
	}

	return onlineFriends, nil
}

// getGamertagFromXUID retrieves the gamertag for a given XUID using the Profile API
func (conf PublishConfig) getGamertagFromXUID(ctx context.Context, xuid string) (string, error) {
	url := fmt.Sprintf("https://profile.xboxlive.com/users/xuid(%s)/profile/settings?settings=Gamertag", xuid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Xbl-Contract-Version", "2")

	resp, err := conf.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		ProfileUsers []struct {
			Settings []struct {
				Value string `json:"value"`
			} `json:"settings"`
		} `json:"profileUsers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(response.ProfileUsers) > 0 && len(response.ProfileUsers[0].Settings) > 0 {
		return response.ProfileUsers[0].Settings[0].Value, nil
	}

	return "", fmt.Errorf("gamertag not found for XUID: %s", xuid)
}
