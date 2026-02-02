package email

// Email templates using HTML

const baseTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background: linear-gradient(135deg, #2563eb, #1d4ed8);
            color: white;
            padding: 30px;
            text-align: center;
            border-radius: 10px 10px 0 0;
        }
        .header h1 {
            margin: 0;
            font-size: 24px;
        }
        .content {
            background: #ffffff;
            padding: 30px;
            border: 1px solid #e5e7eb;
            border-top: none;
        }
        .footer {
            background: #f9fafb;
            padding: 20px;
            text-align: center;
            font-size: 12px;
            color: #6b7280;
            border: 1px solid #e5e7eb;
            border-top: none;
            border-radius: 0 0 10px 10px;
        }
        .button {
            display: inline-block;
            background: #2563eb;
            color: white;
            padding: 12px 30px;
            text-decoration: none;
            border-radius: 6px;
            margin: 20px 0;
        }
        .button:hover {
            background: #1d4ed8;
        }
        .info-box {
            background: #f3f4f6;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
        }
        .info-row {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid #e5e7eb;
        }
        .info-row:last-child {
            border-bottom: none;
        }
        .info-label {
            color: #6b7280;
        }
        .info-value {
            font-weight: 600;
        }
        .highlight {
            color: #2563eb;
            font-weight: 600;
        }
        .warning {
            background: #fef3c7;
            border: 1px solid #f59e0b;
            padding: 15px;
            border-radius: 8px;
            margin: 20px 0;
        }
        .success {
            background: #d1fae5;
            border: 1px solid #10b981;
            padding: 15px;
            border-radius: 8px;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>SIGEC-VE</h1>
        <p style="margin: 5px 0 0 0; opacity: 0.9;">Electric Vehicle Charging Platform</p>
    </div>
    <div class="content">
        {{CONTENT}}
    </div>
    <div class="footer">
        <p>&copy; 2024 SIGEC-VE. All rights reserved.</p>
        <p>This is an automated message. Please do not reply to this email.</p>
    </div>
</body>
</html>
`

const welcomeTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #2563eb, #1d4ed8); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { background: #ffffff; padding: 30px; border: 1px solid #e5e7eb; border-top: none; }
        .footer { background: #f9fafb; padding: 20px; text-align: center; font-size: 12px; color: #6b7280; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 10px 10px; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 30px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .features { margin: 20px 0; }
        .feature { padding: 10px 0; border-bottom: 1px solid #e5e7eb; }
        .feature:last-child { border-bottom: none; }
    </style>
</head>
<body>
    <div class="header">
        <h1>SIGEC-VE</h1>
        <p style="margin: 5px 0 0 0; opacity: 0.9;">Electric Vehicle Charging Platform</p>
    </div>
    <div class="content">
        <h2>Welcome, {{.UserName}}!</h2>
        <p>Thank you for joining SIGEC-VE, your smart electric vehicle charging platform.</p>

        <div class="features">
            <h3>What you can do:</h3>
            <div class="feature">Find nearby charging stations</div>
            <div class="feature">Start and stop charging sessions</div>
            <div class="feature">Track your charging history</div>
            <div class="feature">Use voice commands for hands-free control</div>
            <div class="feature">Monitor costs and energy consumption</div>
        </div>

        <p style="text-align: center;">
            <a href="{{.BaseURL}}/dashboard" class="button">Get Started</a>
        </p>

        <p>If you have any questions, our support team is here to help.</p>
    </div>
    <div class="footer">
        <p>&copy; 2024 SIGEC-VE. All rights reserved.</p>
        <p>This is an automated message. Please do not reply to this email.</p>
    </div>
</body>
</html>
`

const chargingStartedTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #2563eb, #1d4ed8); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { background: #ffffff; padding: 30px; border: 1px solid #e5e7eb; border-top: none; }
        .footer { background: #f9fafb; padding: 20px; text-align: center; font-size: 12px; color: #6b7280; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 10px 10px; }
        .info-box { background: #d1fae5; border: 1px solid #10b981; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .info-row { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #a7f3d0; }
        .info-row:last-child { border-bottom: none; }
        .info-label { color: #047857; }
        .info-value { font-weight: 600; color: #065f46; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 30px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>SIGEC-VE</h1>
        <p style="margin: 5px 0 0 0; opacity: 0.9;">Electric Vehicle Charging Platform</p>
    </div>
    <div class="content">
        <h2>Charging Session Started</h2>
        <p>Hello {{.UserName}},</p>
        <p>Your charging session has started successfully.</p>

        <div class="info-box">
            <div class="info-row">
                <span class="info-label">Transaction ID</span>
                <span class="info-value">{{.TransactionID}}</span>
            </div>
            <div class="info-row">
                <span class="info-label">Station</span>
                <span class="info-value">{{.StationName}}</span>
            </div>
            <div class="info-row">
                <span class="info-label">Start Time</span>
                <span class="info-value">{{.StartTime}}</span>
            </div>
        </div>

        <p>You can monitor your charging session in real-time through the app.</p>

        <p style="text-align: center;">
            <a href="{{.BaseURL}}/transactions/{{.TransactionID}}" class="button">View Session</a>
        </p>
    </div>
    <div class="footer">
        <p>&copy; 2024 SIGEC-VE. All rights reserved.</p>
        <p>This is an automated message. Please do not reply to this email.</p>
    </div>
</body>
</html>
`

const chargingCompletedTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #2563eb, #1d4ed8); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { background: #ffffff; padding: 30px; border: 1px solid #e5e7eb; border-top: none; }
        .footer { background: #f9fafb; padding: 20px; text-align: center; font-size: 12px; color: #6b7280; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 10px 10px; }
        .info-box { background: #f3f4f6; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .info-row { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #e5e7eb; }
        .info-row:last-child { border-bottom: none; }
        .info-label { color: #6b7280; }
        .info-value { font-weight: 600; }
        .total-box { background: #2563eb; color: white; padding: 20px; border-radius: 8px; margin: 20px 0; text-align: center; }
        .total-amount { font-size: 32px; font-weight: bold; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 30px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>SIGEC-VE</h1>
        <p style="margin: 5px 0 0 0; opacity: 0.9;">Electric Vehicle Charging Platform</p>
    </div>
    <div class="content">
        <h2>Charging Session Completed</h2>
        <p>Hello {{.UserName}},</p>
        <p>Your charging session has been completed successfully.</p>

        <div class="info-box">
            <div class="info-row">
                <span class="info-label">Transaction ID</span>
                <span class="info-value">{{.TransactionID}}</span>
            </div>
            <div class="info-row">
                <span class="info-label">Energy Delivered</span>
                <span class="info-value">{{.EnergyKWh}} kWh</span>
            </div>
            <div class="info-row">
                <span class="info-label">Duration</span>
                <span class="info-value">{{.Duration}}</span>
            </div>
        </div>

        <div class="total-box">
            <p style="margin: 0 0 5px 0; opacity: 0.9;">Total Cost</p>
            <div class="total-amount">{{.Currency}} {{.Cost}}</div>
        </div>

        <p>Thank you for using SIGEC-VE!</p>

        <p style="text-align: center;">
            <a href="{{.BaseURL}}/transactions/{{.TransactionID}}" class="button">View Details</a>
        </p>
    </div>
    <div class="footer">
        <p>&copy; 2024 SIGEC-VE. All rights reserved.</p>
        <p>This is an automated message. Please do not reply to this email.</p>
    </div>
</body>
</html>
`

const passwordResetTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #2563eb, #1d4ed8); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { background: #ffffff; padding: 30px; border: 1px solid #e5e7eb; border-top: none; }
        .footer { background: #f9fafb; padding: 20px; text-align: center; font-size: 12px; color: #6b7280; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 10px 10px; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 30px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .warning { background: #fef3c7; border: 1px solid #f59e0b; padding: 15px; border-radius: 8px; margin: 20px 0; color: #92400e; }
    </style>
</head>
<body>
    <div class="header">
        <h1>SIGEC-VE</h1>
        <p style="margin: 5px 0 0 0; opacity: 0.9;">Electric Vehicle Charging Platform</p>
    </div>
    <div class="content">
        <h2>Reset Your Password</h2>
        <p>Hello {{.UserName}},</p>
        <p>We received a request to reset your password. Click the button below to create a new password:</p>

        <p style="text-align: center;">
            <a href="{{.ResetURL}}" class="button">Reset Password</a>
        </p>

        <div class="warning">
            <strong>Security Notice:</strong> This link will expire in 1 hour. If you didn't request a password reset, please ignore this email or contact support if you're concerned about your account security.
        </div>

        <p style="font-size: 12px; color: #6b7280;">
            If the button doesn't work, copy and paste this link into your browser:<br>
            <a href="{{.ResetURL}}" style="color: #2563eb; word-break: break-all;">{{.ResetURL}}</a>
        </p>
    </div>
    <div class="footer">
        <p>&copy; 2024 SIGEC-VE. All rights reserved.</p>
        <p>This is an automated message. Please do not reply to this email.</p>
    </div>
</body>
</html>
`

const invoiceTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #2563eb, #1d4ed8); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { background: #ffffff; padding: 30px; border: 1px solid #e5e7eb; border-top: none; }
        .footer { background: #f9fafb; padding: 20px; text-align: center; font-size: 12px; color: #6b7280; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 10px 10px; }
        .invoice-header { display: flex; justify-content: space-between; margin-bottom: 30px; padding-bottom: 20px; border-bottom: 2px solid #e5e7eb; }
        .invoice-number { font-size: 24px; font-weight: bold; color: #2563eb; }
        .invoice-date { color: #6b7280; }
        .info-box { background: #f3f4f6; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .info-row { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #e5e7eb; }
        .info-row:last-child { border-bottom: none; }
        .info-label { color: #6b7280; }
        .info-value { font-weight: 600; }
        .total-box { background: #1f2937; color: white; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .total-row { display: flex; justify-content: space-between; padding: 8px 0; }
        .total-amount { font-size: 24px; font-weight: bold; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 30px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>SIGEC-VE</h1>
        <p style="margin: 5px 0 0 0; opacity: 0.9;">Electric Vehicle Charging Platform</p>
    </div>
    <div class="content">
        <div class="invoice-header">
            <div>
                <div class="invoice-number">Invoice #{{.InvoiceID}}</div>
                <div class="invoice-date">Date: {{.Date}}</div>
            </div>
        </div>

        <p>Hello {{.UserName}},</p>
        <p>Here is your invoice for the recent charging session:</p>

        <div class="info-box">
            <div class="info-row">
                <span class="info-label">Transaction ID</span>
                <span class="info-value">{{.TransactionID}}</span>
            </div>
            <div class="info-row">
                <span class="info-label">Station</span>
                <span class="info-value">{{.StationName}}</span>
            </div>
            <div class="info-row">
                <span class="info-label">Energy Delivered</span>
                <span class="info-value">{{.EnergyKWh}} kWh</span>
            </div>
            <div class="info-row">
                <span class="info-label">Duration</span>
                <span class="info-value">{{.Duration}}</span>
            </div>
        </div>

        <div class="total-box">
            <div class="total-row">
                <span>Total Amount</span>
                <span class="total-amount">{{.Currency}} {{.Amount}}</span>
            </div>
        </div>

        <p style="text-align: center;">
            <a href="{{.BaseURL}}/invoices/{{.InvoiceID}}" class="button">Download PDF</a>
        </p>
    </div>
    <div class="footer">
        <p>&copy; 2024 SIGEC-VE. All rights reserved.</p>
        <p>This is an automated message. Please do not reply to this email.</p>
    </div>
</body>
</html>
`

const lowBalanceTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #f59e0b, #d97706); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { background: #ffffff; padding: 30px; border: 1px solid #e5e7eb; border-top: none; }
        .footer { background: #f9fafb; padding: 20px; text-align: center; font-size: 12px; color: #6b7280; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 10px 10px; }
        .warning-box { background: #fef3c7; border: 2px solid #f59e0b; padding: 20px; border-radius: 8px; margin: 20px 0; text-align: center; }
        .balance { font-size: 32px; font-weight: bold; color: #d97706; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 30px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>SIGEC-VE</h1>
        <p style="margin: 5px 0 0 0; opacity: 0.9;">Low Balance Warning</p>
    </div>
    <div class="content">
        <h2>Your Balance is Running Low</h2>
        <p>Hello {{.UserName}},</p>
        <p>Your account balance is running low. Please add funds to continue using our charging services without interruption.</p>

        <div class="warning-box">
            <p style="margin: 0 0 10px 0; color: #92400e;">Current Balance</p>
            <div class="balance">{{.Currency}} {{.Balance}}</div>
        </div>

        <p>We recommend maintaining a minimum balance of R$ 50.00 to ensure uninterrupted charging sessions.</p>

        <p style="text-align: center;">
            <a href="{{.BaseURL}}/wallet/add-funds" class="button">Add Funds</a>
        </p>
    </div>
    <div class="footer">
        <p>&copy; 2024 SIGEC-VE. All rights reserved.</p>
        <p>This is an automated message. Please do not reply to this email.</p>
    </div>
</body>
</html>
`
