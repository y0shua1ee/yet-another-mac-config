-- Hammerspoon 配置（性能优先版本）
--
-- 通过尽量少的提示与日志来降低事件 tap 被系统停用的概率。

-- 辅助：Ctrl+Alt+Cmd+R 快速重载配置
hs.hotkey.bind({"cmd", "alt", "ctrl"}, "R", function()
  hs.reload()
end)

--------------------------------------------------------------------------------
-- 1 & 2: 按住 Cmd 双击 W/Q，分别关闭标签页与应用
--------------------------------------------------------------------------------
local cmdIsDown = false
local waitingForSecondW = false
local waitingForSecondQ = false
local wTimer = nil
local qTimer = nil
local wInterval = 0.4 -- W 默认 400ms 内需要第二次
local qInterval = 0.8 -- Q 容许稍长一点，防止误关
local doublePressTap

local function stopTimer(timerRef)
    if timerRef then
        timerRef:stop()
    end
end

local function resetWState()
    stopTimer(wTimer)
    wTimer = nil
    waitingForSecondW = false
end

local function resetQState()
    stopTimer(qTimer)
    qTimer = nil
    waitingForSecondQ = false
end

-- 为避免捕获到自身注入的 Cmd+W/Q，这里注入前暂停 tap
local function injectCmdStroke(key)
    if doublePressTap then
        doublePressTap:stop()
    end
    hs.eventtap.keyStroke({"cmd"}, key, 0)
    if doublePressTap then
        doublePressTap:start()
    end
end

doublePressTap = hs.eventtap.new({hs.eventtap.event.types.keyDown, hs.eventtap.event.types.flagsChanged}, function(e)
    local eventType = e:getType()

    if eventType == hs.eventtap.event.types.flagsChanged then
        local newCmdState = e:getFlags().cmd
        if cmdIsDown ~= newCmdState then
            cmdIsDown = newCmdState
            if not cmdIsDown then
                resetWState()
                resetQState()
            end
        end
        return false
    end

    if eventType ~= hs.eventtap.event.types.keyDown then
        return false
    end

    local keyCode = e:getKeyCode()

    if cmdIsDown and keyCode == hs.keycodes.map.w then
        if not waitingForSecondW then
            waitingForSecondW = true
            wTimer = hs.timer.doAfter(wInterval, resetWState)
            return true
        else
            resetWState()
            injectCmdStroke("w")
            return true
        end
    end

    if cmdIsDown and keyCode == hs.keycodes.map.q then
        if not waitingForSecondQ then
            waitingForSecondQ = true
            qTimer = hs.timer.doAfter(qInterval, resetQState)
            return true
        else
            resetQState()
            injectCmdStroke("q")
            return true
        end
    end

    return false
end)
doublePressTap:start()

--------------------------------------------------------------------------------
-- 3. 将右 Cmd 映射为 F19（供其他工具绑定）
--------------------------------------------------------------------------------
local rightCmdRemapper = hs.eventtap.new({hs.eventtap.event.types.keyDown, hs.eventtap.event.types.keyUp}, function(event)
    if event:getKeyCode() == hs.keycodes.map.rightcmd then
        if event:getType() == hs.eventtap.event.types.keyDown then
            hs.eventtap.keyStroke({}, 'F19')
        end
        return true
    end
    return false
end)
rightCmdRemapper:start()

--------------------------------------------------------------------------------
-- 4. Ctrl + Alt + T 打开 Ghostty
--------------------------------------------------------------------------------
local function openGhosttyInNewWindow()
    local app = hs.application.get("Ghostty")

    if not app then
        -- Ghostty 未运行→启动新实例（必定生成新窗口）
        hs.task.new("/usr/bin/open", nil, {"-na", "Ghostty"}):start()
        return
    end

    -- 已运行→激活后发送 Cmd+N，确保开新窗口，新窗口自带首个标签页
    app:activate()
    hs.timer.doAfter(0.05, function()
        hs.eventtap.keyStroke({"cmd"}, "n", 0, app)
    end)
end

hs.hotkey.bind({"ctrl", "alt"}, "T", openGhosttyInNewWindow)

-- 最后弹出提示，确认配置加载完成
hs.alert.show("Hammerspoon: Performance Version Loaded!")
