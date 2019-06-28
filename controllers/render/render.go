package render

import (
	"github.com/mlogclub/mlog/services/cache"
	"html/template"
	"math/rand"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/mlogclub/simple"
	"github.com/sirupsen/logrus"

	"github.com/mlogclub/mlog/model"
	"github.com/mlogclub/mlog/services"
	"github.com/mlogclub/mlog/utils"
)

var DefaultAvatars = []string{
	"https://file.mlog.club/avatar/club_default_avatar1.png",
	"https://file.mlog.club/avatar/club_default_avatar2.png",
	"https://file.mlog.club/avatar/club_default_avatar3.png",
	"https://file.mlog.club/avatar/club_default_avatar4.png",
	"https://file.mlog.club/avatar/club_default_avatar5.png",
	"https://file.mlog.club/avatar/club_default_avatar6.png",
}

func View(ctx context.Context, filename string, viewModel iris.Map) {
	if viewModel == nil {
		viewModel = iris.Map{}
	}

	// 公用
	viewModel[utils.GlobalFieldCurrentUser] = BuildCurrentUser(ctx)

	err := ctx.View(filename, viewModel)
	if err != nil {
		logrus.Error(err)
	}
}

func GetDefaultAvatar(id int64) string {
	if id <= 0 {
		return DefaultAvatars[0]
	}
	// i := len(DefaultAvatars) % int(id)
	i := rand.Intn(len(DefaultAvatars))
	return DefaultAvatars[i]
}

func BuildUserDefaultIfNull(id int64) *model.UserInfo {
	user := cache.UserCache.Get(id)
	if user == nil {
		user = &model.User{}
		user.Id = id
		user.Username = strconv.FormatInt(id, 10)
		user.Avatar = GetDefaultAvatar(id)
		user.CreateTime = simple.NowTimestamp()
	}
	return BuildUser(user)
}

func BuildCurrentUser(ctx context.Context) *model.UserInfo {
	currentUser := utils.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil
	}
	return BuildUserById(currentUser.Id)
}

func BuildUserById(id int64) *model.UserInfo {
	user := cache.UserCache.Get(id)
	if user == nil {
		return nil
	}
	return BuildUser(user)
}

func BuildUser(user *model.User) *model.UserInfo {
	if user == nil {
		return nil
	}
	avatar := user.Avatar
	if len(avatar) == 0 {
		avatar = GetDefaultAvatar(user.Id)
	}
	roles := strings.Split(user.Roles, ",")
	return &model.UserInfo{
		Id:          user.Id,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Avatar:      avatar,
		Email:       user.Email,
		Type:        user.Type,
		Roles:       roles,
		Description: user.Description,
		CreateTime:  user.CreateTime,
	}
}

func BuildUsers(users []model.User) []model.UserInfo {
	if len(users) == 0 {
		return nil
	}
	var responses []model.UserInfo
	for _, user := range users {
		item := BuildUser(&user)
		if item != nil {
			responses = append(responses, *item)
		}
	}
	return responses
}

func BuildCategory(categoryId int64) *model.CategoryResponse {
	if categoryId > 0 {
		cat := cache.CategoryCache.Get(categoryId)
		if cat != nil {
			return &model.CategoryResponse{CategoryId: cat.Id, CategoryName: cat.Name}
		}
	}
	return nil
}

func BuildArticle(article *model.Article) *model.ArticleResponse {
	if article == nil {
		return nil
	}

	rsp := &model.ArticleResponse{}

	rsp.ArticleId = article.Id
	rsp.Title = article.Title
	rsp.Summary = article.Summary
	rsp.SourceUrl = article.SourceUrl
	rsp.CreateTime = article.CreateTime

	rsp.User = BuildUserDefaultIfNull(article.UserId)
	rsp.Category = BuildCategory(article.CategoryId)

	tagIds := cache.ArticleTagCache.Get(article.Id)
	tags := cache.TagCache.GetList(tagIds)
	rsp.Tags = BuildTags(tags)

	if article.ContentType == model.ArticleContentTypeMarkdown {
		mr := simple.Markdown(article.Content)
		rsp.Content = template.HTML(BuildHtmlContent(mr.ContentHtml))
		if len(rsp.Summary) == 0 {
			rsp.Summary = mr.SummaryText
		}
	} else {
		rsp.Content = template.HTML(BuildHtmlContent(article.Content))
		if len(rsp.Summary) == 0 {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
			if err == nil {
				text := doc.Text()
				rsp.Summary = simple.GetSummary(text, 256)
			}
		}
	}

	return rsp
}

func BuildArticles(articles []model.Article) []model.ArticleResponse {
	if articles == nil || len(articles) == 0 {
		return nil
	}
	var responses []model.ArticleResponse
	for _, article := range articles {
		responses = append(responses, *BuildArticle(&article))
	}
	return responses
}

func BuildTopic(topic *model.Topic) *model.TopicResponse {
	if topic == nil {
		return nil
	}

	rsp := &model.TopicResponse{}

	rsp.TopicId = topic.Id
	rsp.Title = topic.Title
	rsp.User = BuildUserDefaultIfNull(topic.UserId)
	rsp.LastCommentTime = topic.LastCommentTime
	rsp.CreateTime = topic.CreateTime
	rsp.ViewCount = topic.ViewCount

	tags := services.TopicServiceInstance.GetTopicTags(topic.Id)
	rsp.Tags = BuildTags(tags)

	// tagIds := cache.ArticleTagCache.Get(article.Id)
	// tags := cache.TagCache.GetList(tagIds)
	// rsp.Tags = BuildTags(tags)

	mr := simple.Markdown(topic.Content)
	rsp.Content = template.HTML(BuildHtmlContent(mr.ContentHtml))

	return rsp
}

func BuildTopics(topics []model.Topic) []model.TopicResponse {
	if topics == nil || len(topics) == 0 {
		return nil
	}
	var responses []model.TopicResponse
	for _, topic := range topics {
		responses = append(responses, *BuildTopic(&topic))
	}
	return responses
}

func BuildComment(comment model.Comment) *model.CommentResponse {
	return _buildComment(&comment, true)
}

func _buildComment(comment *model.Comment, buildQuote bool) *model.CommentResponse {
	if comment == nil {
		return nil
	}
	markdownResult := simple.Markdown(comment.Content)
	content := template.HTML(markdownResult.ContentHtml)

	ret := &model.CommentResponse{
		CommentId:  comment.Id,
		User:       BuildUserDefaultIfNull(comment.UserId),
		EntityType: comment.EntityType,
		EntityId:   comment.EntityId,
		Content:    content,
		QuoteId:    comment.QuoteId,
		Status:     comment.Status,
		CreateTime: comment.CreateTime,
	}

	if buildQuote && comment.QuoteId > 0 {
		quote := _buildComment(services.CommentServiceInstance.Get(comment.QuoteId), false)
		if quote != nil {
			ret.Quote = quote
			ret.QuoteContent = template.HTML(quote.User.Nickname+"：") + quote.Content
		}
	}
	return ret
}

func BuildTag(tag *model.Tag) *model.TagResponse {
	if tag == nil {
		return nil
	}
	return &model.TagResponse{TagId: tag.Id, TagName: tag.Name}
}

func BuildTags(tags []model.Tag) *[]model.TagResponse {
	if len(tags) == 0 {
		return nil
	}
	var responses []model.TagResponse
	for _, tag := range tags {
		responses = append(responses, *BuildTag(&tag))
	}
	return &responses
}

func BuildFavorite(favorite *model.Favorite) *model.FavoriteResponse {
	rsp := &model.FavoriteResponse{}
	rsp.FavoriteId = favorite.Id
	rsp.EntityType = favorite.EntityType
	rsp.CreateTime = favorite.CreateTime

	if favorite.EntityType == model.EntityTypeArticle {
		article := services.ArticleServiceInstance.Get(favorite.EntityId)
		if article == nil || article.Status != model.ArticleStatusPublished {
			rsp.Deleted = true
		} else {
			rsp.Url = utils.BuildArticleUrl(article.Id)
			rsp.User = BuildUserById(article.UserId)
			rsp.Title = article.Title
			if article.ContentType == model.ArticleContentTypeMarkdown {
				rsp.Content = utils.GetMarkdownSummary(article.Content)
			} else {
				doc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
				if err == nil {
					text := doc.Text()
					rsp.Content = simple.GetSummary(text, 256)
				}
			}
		}
	} else {
		topic := services.TopicServiceInstance.Get(favorite.EntityId)
		if topic == nil || topic.Status != model.TopicStatusOk {
			rsp.Deleted = true
		} else {
			rsp.Url = utils.BuildTopicUrl(topic.Id)
			rsp.User = BuildUserById(topic.UserId)
			rsp.Title = topic.Title
			rsp.Content = utils.GetMarkdownSummary(topic.Content)
		}
	}
	return rsp
}

func BuildFavorites(favorites []model.Favorite) []model.FavoriteResponse {
	if favorites == nil || len(favorites) == 0 {
		return nil
	}
	var responses []model.FavoriteResponse
	for _, favorite := range favorites {
		responses = append(responses, *BuildFavorite(&favorite))
	}
	return responses
}

func BuildMessage(message *model.Message) *model.MessageResponse {
	if message == nil {
		return nil
	}
	user := BuildUserDefaultIfNull(message.UserId)
	return &model.MessageResponse{
		MessageId:    message.Id,
		UserId:       message.UserId,
		User:         user,
		Content:      message.Content,
		QuoteContent: message.QuoteContent,
		Type:         message.Type,
		ExtraData:    message.ExtraData,
		Status:       message.Status,
		CreateTime:   message.CreateTime,
	}
}

func BuildMessages(messages []model.Message) []model.MessageResponse {
	if len(messages) == 0 {
		return nil
	}
	var responses []model.MessageResponse
	for _, message := range messages {
		responses = append(responses, *BuildMessage(&message))
	}
	return responses
}

func BuildHtmlContent(htmlContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	doc.Find("a").Each(func(i int, selection *goquery.Selection) {
		href := selection.AttrOr("href", "")

		// // 标记站外链接，搜索引擎爬虫不传递权重值
		// if !strings.Contains(href, "mlog.club") {
		// 	selection.SetAttr("rel", "external nofollow")
		// }

		// 内部跳转
		if len(href) > 0 && !strings.Contains(href, "mlog.club") {
			v := url.Values{}
			v.Add("url", href)
			u, _ := url.Parse(utils.BuildAbsUrl("/redirect"))
			u.RawQuery = v.Encode()
			selection.SetAttr("href", u.String())
			selection.SetAttr("target", "_blank")
		}

		// 如果a标签没有title，那么设置title
		title := selection.AttrOr("title", "")
		if len(title) == 0 {
			selection.SetAttr("title", selection.Text())
		}
	})
	html, err := doc.Html()
	if err != nil {
		return htmlContent
	}
	return html
}