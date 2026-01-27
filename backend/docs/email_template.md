# Email Templates

Email templates support variable rendering using Go template syntax, for example:

- {{ .user.username }}
- {{ .order.no }}
- {{ .vps.expire_at }}
- {{ .message }}

HTML is supported. If the template body contains HTML tags, it is sent as HTML; otherwise it is sent as plain text.

## Built-in variables
- user.id
- user.username
- user.email
- user.qq
- order.no
- vps.name
- vps.expire_at
- message

## API Endpoints

### List email templates
```
GET /admin/api/v1/email-templates
Authorization: Bearer <admin_jwt>
```

Response:
```json
{
  "items": [
    {
      "id": 1,
      "name": "provision_success",
      "subject": "VPS Provisioned: Order {{ .order.no }}",
      "body": "<html>...</html>",
      "enabled": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Create email template
```
POST /admin/api/v1/email-templates
Authorization: Bearer <admin_jwt>
Content-Type: application/json
```

Request body:
```json
{
  "name": "custom_template",
  "subject": "Custom Subject",
  "body": "Template content with {{ .user.username }}",
  "enabled": true
}
```

### Update email template
```
PATCH /admin/api/v1/email-templates/:id
Authorization: Bearer <admin_jwt>
Content-Type: application/json
```

Request body (id is taken from URL parameter):
```json
{
  "name": "custom_template",
  "subject": "Updated Subject",
  "body": "Updated content",
  "enabled": false
}
```

Note: The `id` field in the request body is ignored. The template ID is taken from the URL path parameter `:id`.

### Delete email template
```
DELETE /admin/api/v1/email-templates/:id
Authorization: Bearer <admin_jwt>
```

## SMTP test
Use the admin endpoint to render and send a test email:

```
curl -X POST http://localhost:8080/admin/api/v1/integrations/smtp/test \
  -H "Authorization: Bearer <admin_jwt>" \
  -H "Content-Type: application/json" \
  -d '{"to":"test@example.com","template_name":"order_approved","variables":{"user":{"username":"demo"},"order":{"no":"ORD-001"},"message":"Approved"}}'
```

## Example HTML template

Subject:

```
Order Approved: {{ .order.no }}
```

Body:

```
<!DOCTYPE html>
<html>
<body>
  <h2>Order Approved</h2>
  <p>Hi {{ .user.username }},</p>
  <p>Your order {{ .order.no }} has been approved.</p>
  <p>{{ .message }}</p>
</body>
</html>
```
