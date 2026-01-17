-- Migration: Add server name update functionality
-- Created: 2025-01-17
-- Description: Add foreign key relationship and server name update function

-- Add foreign key constraint to user_servers table
ALTER TABLE user_servers 
ADD CONSTRAINT fk_user_servers_server_id 
FOREIGN KEY (server_id) REFERENCES servers(server_id) ON DELETE CASCADE;

-- Create function to update server name
CREATE OR REPLACE FUNCTION update_server_name(
    p_user_id INTEGER,
    p_server_id VARCHAR(255),
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
    WHERE server_id = p_server_id;
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Create function to get server with user permissions
CREATE OR REPLACE FUNCTION get_user_server(
    p_user_id INTEGER,
    p_server_id VARCHAR(255
) RETURNS TABLE (
    server_id VARCHAR(255),
    name VARCHAR(255),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    role VARCHAR(50),
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
        us.role,
        us.added_at
    FROM servers s
    INNER JOIN user_servers us ON s.server_id = us.server_id
    WHERE us.user_id = p_user_id
    AND s.server_id = p_server_id;
END;
$$ LANGUAGE plpgsql;
