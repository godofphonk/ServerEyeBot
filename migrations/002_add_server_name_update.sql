-- Migration: Add server name update functionality
-- Created: 2025-01-17
-- Description: Add functions for server management

-- Create function to update server name
CREATE OR REPLACE FUNCTION update_server_name(
    p_user_id INTEGER,
    p_server_id INTEGER,
    p_new_name VARCHAR(255)
) RETURNS BOOLEAN AS $$
DECLARE
    server_exists BOOLEAN;
BEGIN
    -- Check if user has access to this server
    SELECT EXISTS(
        SELECT 1 FROM user_servers 
        WHERE user_id = p_user_id 
        AND server_id = p_server_id
    ) INTO server_exists;
    
    IF NOT server_exists THEN
        RETURN FALSE;
    END IF;
    
    -- Update server name
    UPDATE servers 
    SET name = p_new_name 
    WHERE id = p_server_id;
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Create function to get server with user permissions
CREATE OR REPLACE FUNCTION get_user_server(
    p_user_id INTEGER,
    p_server_id INTEGER
) RETURNS TABLE (
    server_id VARCHAR(255),
    name VARCHAR(255),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    server_key VARCHAR(255),
    is_monitoring BOOLEAN,
    added_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.server_id,
        s.name,
        s.description,
        s.created_at,
        s.updated_at,
        us.server_key,
        us.is_monitoring,
        us.created_at as added_at
    FROM servers s
    INNER JOIN user_servers us ON s.id = us.server_id
    WHERE us.user_id = p_user_id
    AND s.id = p_server_id;
END;
$$ LANGUAGE plpgsql;
